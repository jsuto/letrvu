package imap

import (
	"strings"
	"testing"

	goimap "github.com/emersion/go-imap/v2"
)

// --- folderPriority ----------------------------------------------------------

func TestFolderPriority_KnownFolders(t *testing.T) {
	cases := []struct {
		name string
		want int
	}{
		{"INBOX", 0},
		{"Inbox", 0},
		{"inbox", 0},
		{"Drafts", 1},
		{"Draft", 1},
		{"drafts", 1},
		{"Sent", 2},
		{"Sent Items", 2},
		{"Sent Mail", 2},
		{"Junk", 3},
		{"Junk Email", 3},
		{"Spam", 3},
		{"Trash", 4},
		{"Deleted", 4},
		{"Deleted Items", 4},
		{"Archives", 5},
		{"Work", 5},
		{"", 5},
	}
	for _, tc := range cases {
		got := folderPriority(tc.name)
		if got != tc.want {
			t.Errorf("folderPriority(%q) = %d, want %d", tc.name, got, tc.want)
		}
	}
}

func TestFolderPriority_SortOrder(t *testing.T) {
	// Verify that priority values produce the right ordering.
	ordered := []string{"INBOX", "Drafts", "Sent", "Junk", "Trash", "Aardvark"}
	for i := 1; i < len(ordered); i++ {
		prev := folderPriority(ordered[i-1])
		curr := folderPriority(ordered[i])
		if prev > curr {
			t.Errorf("folderPriority(%q)=%d should be <= folderPriority(%q)=%d",
				ordered[i-1], prev, ordered[i], curr)
		}
	}
}

// --- parseMIMEBody -----------------------------------------------------------

func plainTextEmail(subject, body string) []byte {
	return []byte("From: sender@example.com\r\n" +
		"To: rcpt@example.com\r\n" +
		"Subject: " + subject + "\r\n" +
		"MIME-Version: 1.0\r\n" +
		"Content-Type: text/plain; charset=UTF-8\r\n" +
		"\r\n" +
		body)
}

func multipartEmail(text, html string) []byte {
	const boundary = "test-boundary-001"
	return []byte("From: sender@example.com\r\n" +
		"To: rcpt@example.com\r\n" +
		"Subject: test\r\n" +
		"MIME-Version: 1.0\r\n" +
		"Content-Type: multipart/alternative; boundary=\"" + boundary + "\"\r\n" +
		"\r\n" +
		"--" + boundary + "\r\n" +
		"Content-Type: text/plain; charset=UTF-8\r\n" +
		"\r\n" +
		text + "\r\n" +
		"--" + boundary + "\r\n" +
		"Content-Type: text/html; charset=UTF-8\r\n" +
		"\r\n" +
		html + "\r\n" +
		"--" + boundary + "--\r\n")
}

func TestParseMIMEBody_PlainText(t *testing.T) {
	full := &MessageFull{}
	if err := parseMIMEBody(plainTextEmail("hi", "Hello, world!"), full); err != nil {
		t.Fatalf("parseMIMEBody: %v", err)
	}
	if full.TextBody != "Hello, world!" {
		t.Errorf("TextBody = %q, want %q", full.TextBody, "Hello, world!")
	}
	if full.HTMLBody != "" {
		t.Errorf("HTMLBody should be empty, got %q", full.HTMLBody)
	}
}

func TestParseMIMEBody_HTML(t *testing.T) {
	full := &MessageFull{}
	raw := multipartEmail("plain part", "<b>html part</b>")
	if err := parseMIMEBody(raw, full); err != nil {
		t.Fatalf("parseMIMEBody: %v", err)
	}
	if full.TextBody != "plain part" {
		t.Errorf("TextBody = %q, want %q", full.TextBody, "plain part")
	}
	if full.HTMLBody != "<b>html part</b>" {
		t.Errorf("HTMLBody = %q, want %q", full.HTMLBody, "<b>html part</b>")
	}
}

func TestParseMIMEBody_EmptyBody(t *testing.T) {
	full := &MessageFull{}
	if err := parseMIMEBody(plainTextEmail("empty", ""), full); err != nil {
		t.Fatalf("parseMIMEBody: %v", err)
	}
	if full.TextBody != "" {
		t.Errorf("TextBody = %q, want empty", full.TextBody)
	}
}

func TestParseMIMEBody_Attachment(t *testing.T) {
	const boundary = "att-boundary"
	raw := []byte("From: a@b.com\r\n" +
		"MIME-Version: 1.0\r\n" +
		"Content-Type: multipart/mixed; boundary=\"" + boundary + "\"\r\n" +
		"\r\n" +
		"--" + boundary + "\r\n" +
		"Content-Type: text/plain; charset=UTF-8\r\n" +
		"\r\n" +
		"see attachment\r\n" +
		"--" + boundary + "\r\n" +
		"Content-Type: application/pdf\r\n" +
		"Content-Disposition: attachment; filename=\"doc.pdf\"\r\n" +
		"\r\n" +
		"PDFDATA\r\n" +
		"--" + boundary + "--\r\n")

	full := &MessageFull{}
	if err := parseMIMEBody(raw, full); err != nil {
		t.Fatalf("parseMIMEBody: %v", err)
	}
	if full.TextBody != "see attachment" {
		t.Errorf("TextBody = %q", full.TextBody)
	}
	if len(full.Attachments) != 1 {
		t.Fatalf("want 1 attachment, got %d", len(full.Attachments))
	}
	if full.Attachments[0].Filename != "doc.pdf" {
		t.Errorf("Filename = %q, want %q", full.Attachments[0].Filename, "doc.pdf")
	}
	if full.Attachments[0].ContentType != "application/pdf" {
		t.Errorf("ContentType = %q", full.Attachments[0].ContentType)
	}
	if full.Attachments[0].Index != 0 {
		t.Errorf("Index = %d, want 0", full.Attachments[0].Index)
	}
}

func TestParseMIMEBody_CalendarInvite(t *testing.T) {
	const boundary = "cal-boundary"
	ical := "BEGIN:VCALENDAR\r\nEND:VCALENDAR"
	raw := []byte("From: a@b.com\r\n" +
		"MIME-Version: 1.0\r\n" +
		"Content-Type: multipart/mixed; boundary=\"" + boundary + "\"\r\n" +
		"\r\n" +
		"--" + boundary + "\r\n" +
		"Content-Type: text/calendar; charset=UTF-8\r\n" +
		"\r\n" +
		ical + "\r\n" +
		"--" + boundary + "--\r\n")

	full := &MessageFull{}
	if err := parseMIMEBody(raw, full); err != nil {
		t.Fatalf("parseMIMEBody: %v", err)
	}
	if !strings.Contains(full.ICalInvite, "VCALENDAR") {
		t.Errorf("ICalInvite = %q, want VCALENDAR content", full.ICalInvite)
	}
}

// --- pickSpecialFolder -------------------------------------------------------

func makeMailbox(name string, attrs ...goimap.MailboxAttr) *goimap.ListData {
	return &goimap.ListData{Mailbox: name, Attrs: attrs}
}

func TestPickSpecialFolder_ByAttribute(t *testing.T) {
	mboxes := []*goimap.ListData{
		makeMailbox("INBOX"),
		makeMailbox("MyDrafts", goimap.MailboxAttrDrafts),
		makeMailbox("MySent", goimap.MailboxAttrSent),
	}
	got := pickSpecialFolder(mboxes, goimap.MailboxAttrDrafts, []string{"drafts"})
	if got != "MyDrafts" {
		t.Errorf("want MyDrafts, got %q", got)
	}
}

func TestPickSpecialFolder_SentAttribute(t *testing.T) {
	mboxes := []*goimap.ListData{
		makeMailbox("INBOX"),
		makeMailbox("Sent Items", goimap.MailboxAttrSent),
	}
	got := pickSpecialFolder(mboxes, goimap.MailboxAttrSent, []string{"sent", "sent items", "sent mail"})
	if got != "Sent Items" {
		t.Errorf("want %q, got %q", "Sent Items", got)
	}
}

func TestPickSpecialFolder_FallbackByName(t *testing.T) {
	// Server reports no special-use attributes — fall back to name matching.
	mboxes := []*goimap.ListData{
		makeMailbox("INBOX"),
		makeMailbox("Drafts"),
		makeMailbox("Sent"),
	}
	got := pickSpecialFolder(mboxes, goimap.MailboxAttrDrafts, []string{"drafts", "draft"})
	if got != "Drafts" {
		t.Errorf("want Drafts, got %q", got)
	}
}

func TestPickSpecialFolder_FallbackNameCaseInsensitive(t *testing.T) {
	mboxes := []*goimap.ListData{
		makeMailbox("SENT MAIL"),
	}
	got := pickSpecialFolder(mboxes, goimap.MailboxAttrSent, []string{"sent", "sent items", "sent mail"})
	if got != "SENT MAIL" {
		t.Errorf("want %q, got %q", "SENT MAIL", got)
	}
}

func TestPickSpecialFolder_AttributeTakesPriorityOverName(t *testing.T) {
	// "Outbox" has the \Sent attribute; "Sent" is a plain name match.
	// The attribute winner should be returned.
	mboxes := []*goimap.ListData{
		makeMailbox("Sent"),
		makeMailbox("Outbox", goimap.MailboxAttrSent),
	}
	got := pickSpecialFolder(mboxes, goimap.MailboxAttrSent, []string{"sent"})
	if got != "Outbox" {
		t.Errorf("want Outbox (attribute winner), got %q", got)
	}
}

func TestPickSpecialFolder_NoneFound(t *testing.T) {
	mboxes := []*goimap.ListData{
		makeMailbox("INBOX"),
		makeMailbox("Archive"),
	}
	got := pickSpecialFolder(mboxes, goimap.MailboxAttrDrafts, []string{"drafts", "draft"})
	if got != "" {
		t.Errorf("want empty string, got %q", got)
	}
}

func TestPickSpecialFolder_EmptyMailboxList(t *testing.T) {
	got := pickSpecialFolder(nil, goimap.MailboxAttrSent, []string{"sent"})
	if got != "" {
		t.Errorf("want empty string for empty list, got %q", got)
	}
}

// --- bodyHasAttachments ------------------------------------------------------

func TestBodyHasAttachments_SinglePlain(t *testing.T) {
	bs := &goimap.BodyStructureSinglePart{
		Type:    "TEXT",
		Subtype: "PLAIN",
	}
	if bodyHasAttachments(bs) {
		t.Error("plain text part should not have attachments")
	}
}

func TestBodyHasAttachments_AttachmentDisposition(t *testing.T) {
	bs := &goimap.BodyStructureSinglePart{
		Type:    "APPLICATION",
		Subtype: "PDF",
		Extended: &goimap.BodyStructureSinglePartExt{
			Disposition: &goimap.BodyStructureDisposition{Value: "attachment"},
		},
	}
	if !bodyHasAttachments(bs) {
		t.Error("PDF attachment disposition should be detected")
	}
}

func TestBodyHasAttachments_CaseInsensitive(t *testing.T) {
	bs := &goimap.BodyStructureSinglePart{
		Type:    "APPLICATION",
		Subtype: "PDF",
		Extended: &goimap.BodyStructureSinglePartExt{
			Disposition: &goimap.BodyStructureDisposition{Value: "ATTACHMENT"},
		},
	}
	if !bodyHasAttachments(bs) {
		t.Error("attachment detection should be case-insensitive")
	}
}

func TestBodyHasAttachments_Multipart(t *testing.T) {
	bs := &goimap.BodyStructureMultiPart{
		Subtype: "MIXED",
		Children: []goimap.BodyStructure{
			&goimap.BodyStructureSinglePart{Type: "TEXT", Subtype: "PLAIN"},
			&goimap.BodyStructureSinglePart{
				Type:    "APPLICATION",
				Subtype: "PDF",
				Extended: &goimap.BodyStructureSinglePartExt{
					Disposition: &goimap.BodyStructureDisposition{Value: "attachment"},
				},
			},
		},
	}
	if !bodyHasAttachments(bs) {
		t.Error("multipart/mixed with attachment part should be detected")
	}
}

func TestBodyHasAttachments_MultipartNoAttachment(t *testing.T) {
	bs := &goimap.BodyStructureMultiPart{
		Subtype: "ALTERNATIVE",
		Children: []goimap.BodyStructure{
			&goimap.BodyStructureSinglePart{Type: "TEXT", Subtype: "PLAIN"},
			&goimap.BodyStructureSinglePart{Type: "TEXT", Subtype: "HTML"},
		},
	}
	if bodyHasAttachments(bs) {
		t.Error("multipart/alternative with no attachments should return false")
	}
}

// --- formatAddress -----------------------------------------------------------

func TestFormatAddress_WithName(t *testing.T) {
	addr := goimap.Address{Name: "Alice", Mailbox: "alice", Host: "example.com"}
	got := formatAddress(addr)
	want := "Alice <alice@example.com>"
	if got != want {
		t.Errorf("formatAddress = %q, want %q", got, want)
	}
}

func TestFormatAddress_NoName(t *testing.T) {
	addr := goimap.Address{Name: "", Mailbox: "alice", Host: "example.com"}
	got := formatAddress(addr)
	want := "alice@example.com"
	if got != want {
		t.Errorf("formatAddress = %q, want %q", got, want)
	}
}
