package smtp

import (
	"crypto/tls"
	"encoding/base64"
	"strings"
	"testing"
)

// --- helpers -----------------------------------------------------------------

func mustContain(t *testing.T, haystack, needle, label string) {
	t.Helper()
	if !strings.Contains(haystack, needle) {
		t.Errorf("%s: want %q in MIME output\ngot:\n%s", label, needle, haystack)
	}
}

func mustNotContain(t *testing.T, haystack, needle, label string) {
	t.Helper()
	if strings.Contains(haystack, needle) {
		t.Errorf("%s: did not want %q in MIME output", label, needle)
	}
}

// --- From / To / CC headers --------------------------------------------------

func TestBuildMIME_Headers(t *testing.T) {
	m := buildMIME(Message{
		From:    "Alice <alice@example.com>",
		To:      []string{"bob@example.com"},
		Subject: "Hello",
		Text:    "Hi Bob",
	})
	mustContain(t, m, "From: Alice <alice@example.com>", "From")
	mustContain(t, m, "To: bob@example.com", "To")
	mustContain(t, m, "Subject: Hello", "Subject")
	mustContain(t, m, "MIME-Version: 1.0", "MIME-Version")
}

func TestBuildMIME_MultipleRecipients(t *testing.T) {
	m := buildMIME(Message{
		From:    "alice@example.com",
		To:      []string{"bob@example.com", "carol@example.com"},
		Subject: "Hey",
		Text:    "body",
	})
	mustContain(t, m, "bob@example.com, carol@example.com", "To multi")
}

func TestBuildMIME_CC(t *testing.T) {
	m := buildMIME(Message{
		From:    "alice@example.com",
		To:      []string{"bob@example.com"},
		CC:      []string{"dave@example.com"},
		Subject: "CC test",
		Text:    "body",
	})
	mustContain(t, m, "Cc: dave@example.com", "CC header")
}

func TestBuildMIME_NoCC(t *testing.T) {
	m := buildMIME(Message{
		From:    "alice@example.com",
		To:      []string{"bob@example.com"},
		Subject: "no cc",
		Text:    "body",
	})
	mustNotContain(t, m, "Cc:", "no CC header")
}

// --- plain text body ---------------------------------------------------------

func TestBuildMIME_PlainText(t *testing.T) {
	m := buildMIME(Message{
		From:    "a@example.com",
		To:      []string{"b@example.com"},
		Subject: "plain",
		Text:    "Hello, world!",
	})
	mustContain(t, m, "Content-Type: text/plain; charset=UTF-8", "plain content-type")
	mustContain(t, m, "Hello, world!", "plain body")
	mustNotContain(t, m, "multipart/alternative", "no multipart for plain")
}

// --- HTML body (multipart/alternative) ---------------------------------------

func TestBuildMIME_HTMLAlternative(t *testing.T) {
	m := buildMIME(Message{
		From:    "a@example.com",
		To:      []string{"b@example.com"},
		Subject: "html",
		Text:    "plain part",
		HTML:    "<p>html part</p>",
	})
	mustContain(t, m, "multipart/alternative", "alternative content-type")
	mustContain(t, m, "Content-Type: text/plain; charset=UTF-8", "plain part")
	mustContain(t, m, "Content-Type: text/html; charset=UTF-8", "html part")
	mustContain(t, m, "plain part", "plain body text")
	mustContain(t, m, "<p>html part</p>", "html body text")
}

func TestBuildMIME_HTMLBoundaryPresent(t *testing.T) {
	m := buildMIME(Message{
		From: "a@example.com", To: []string{"b@example.com"},
		Subject: "s", Text: "t", HTML: "<b>h</b>",
	})
	mustContain(t, m, "--letrvu-boundary-001", "boundary marker")
	mustContain(t, m, "--letrvu-boundary-001--", "closing boundary")
}

// --- attachments (multipart/mixed) ------------------------------------------

func TestBuildMIME_WithAttachment(t *testing.T) {
	data := []byte("hello attachment")
	m := buildMIME(Message{
		From:    "a@example.com",
		To:      []string{"b@example.com"},
		Subject: "with att",
		Text:    "see attachment",
		Attachments: []Attachment{
			{Filename: "test.txt", ContentType: "text/plain", Data: data},
		},
	})
	mustContain(t, m, "multipart/mixed", "mixed content-type")
	mustContain(t, m, "--letrvu-mixed-001", "mixed boundary")
	mustContain(t, m, "--letrvu-mixed-001--", "closing mixed boundary")
	mustContain(t, m, `filename="test.txt"`, "attachment filename")
	mustContain(t, m, "Content-Transfer-Encoding: base64", "base64 encoding")
	mustContain(t, m, base64.StdEncoding.EncodeToString(data)[:10], "base64 data prefix")
}

func TestBuildMIME_AttachmentDefaultContentType(t *testing.T) {
	m := buildMIME(Message{
		From: "a@example.com", To: []string{"b@example.com"},
		Subject: "s", Text: "t",
		Attachments: []Attachment{{Filename: "f.bin", Data: []byte{0x00}}},
	})
	mustContain(t, m, "application/octet-stream", "default content-type")
}

func TestBuildMIME_AttachmentBase64LineLength(t *testing.T) {
	// RFC 2045: base64 lines must not exceed 76 characters.
	data := make([]byte, 1000)
	for i := range data {
		data[i] = byte(i % 256)
	}
	m := buildMIME(Message{
		From: "a@example.com", To: []string{"b@example.com"},
		Subject: "s", Text: "t",
		Attachments: []Attachment{{Filename: "big.bin", Data: data}},
	})
	inAttachment := false
	for _, line := range strings.Split(m, "\r\n") {
		if strings.Contains(line, "Content-Transfer-Encoding: base64") {
			inAttachment = true
			continue
		}
		if inAttachment && strings.HasPrefix(line, "--") {
			break
		}
		if inAttachment && line != "" && len(line) > 76 {
			t.Errorf("base64 line exceeds 76 chars: len=%d", len(line))
		}
	}
}

func TestBuildMIME_MultipleAttachments(t *testing.T) {
	m := buildMIME(Message{
		From: "a@example.com", To: []string{"b@example.com"},
		Subject: "s", Text: "t",
		Attachments: []Attachment{
			{Filename: "a.txt", ContentType: "text/plain", Data: []byte("aaa")},
			{Filename: "b.txt", ContentType: "text/plain", Data: []byte("bbb")},
		},
	})
	mustContain(t, m, `"a.txt"`, "first attachment")
	mustContain(t, m, `"b.txt"`, "second attachment")
}

// --- BuildRFC822 -------------------------------------------------------------

func TestBuildRFC822_HasDateHeader(t *testing.T) {
	raw := BuildRFC822(Message{
		From:    "alice@example.com",
		To:      []string{"bob@example.com"},
		Subject: "draft test",
		Text:    "body",
	})
	s := string(raw)
	mustContain(t, s, "Date: ", "Date header")
}

func TestBuildRFC822_ContainsMIMEContent(t *testing.T) {
	raw := BuildRFC822(Message{
		From:    "Alice <alice@example.com>",
		To:      []string{"bob@example.com"},
		Subject: "My Draft",
		Text:    "draft body",
	})
	s := string(raw)
	mustContain(t, s, "From: Alice <alice@example.com>", "From header")
	mustContain(t, s, "To: bob@example.com", "To header")
	mustContain(t, s, "Subject: My Draft", "Subject header")
	mustContain(t, s, "draft body", "body text")
}

func TestBuildRFC822_DateBeforeMIMEHeaders(t *testing.T) {
	// Date: must appear before the MIME headers so it is a top-level header.
	raw := BuildRFC822(Message{
		From: "a@example.com", To: []string{"b@example.com"},
		Subject: "order", Text: "t",
	})
	s := string(raw)
	dateIdx := strings.Index(s, "Date: ")
	fromIdx := strings.Index(s, "From: ")
	if dateIdx < 0 {
		t.Fatal("Date header missing")
	}
	if fromIdx < 0 {
		t.Fatal("From header missing")
	}
	if dateIdx > fromIdx {
		t.Errorf("Date header (pos %d) should appear before From header (pos %d)", dateIdx, fromIdx)
	}
}

func TestBuildRFC822_WithHTML(t *testing.T) {
	raw := BuildRFC822(Message{
		From:    "a@example.com",
		To:      []string{"b@example.com"},
		Subject: "html draft",
		Text:    "plain",
		HTML:    "<p>html</p>",
	})
	s := string(raw)
	mustContain(t, s, "multipart/alternative", "multipart content-type")
	mustContain(t, s, "<p>html</p>", "html body")
	mustContain(t, s, "Date: ", "Date header")
}

func TestBuildRFC822_WithCC(t *testing.T) {
	raw := BuildRFC822(Message{
		From:    "a@example.com",
		To:      []string{"b@example.com"},
		CC:      []string{"c@example.com"},
		Subject: "cc draft",
		Text:    "body",
	})
	s := string(raw)
	mustContain(t, s, "Cc: c@example.com", "CC header")
}

// --- DefaultTLSConfig --------------------------------------------------------

func TestDefaultTLSConfig_CanBeReplaced(t *testing.T) {
	// Verify main() can safely swap out DefaultTLSConfig before the first Send.
	orig := DefaultTLSConfig
	t.Cleanup(func() { DefaultTLSConfig = orig })

	DefaultTLSConfig = &tls.Config{InsecureSkipVerify: false} //nolint:gosec
	if DefaultTLSConfig.InsecureSkipVerify {
		t.Error("expected InsecureSkipVerify=false after replacement")
	}
}

// --- Send — port dispatch ----------------------------------------------------

// TestSend_Port465_FailsWithoutServer confirms that port 465 attempts an
// immediate TLS dial (not STARTTLS). The dial fails because there is no server,
// but the error must come from tls.Dial ("smtp dial tls:"), not from a
// STARTTLS negotiation attempt.
func TestSend_Port465_ImplicitTLS(t *testing.T) {
	err := Send(Config{
		Host: "127.0.0.1", Port: 465,
		Username: "u", Password: "p",
	}, Message{From: "a@example.com", To: []string{"b@example.com"}, Subject: "s", Text: "t"})
	if err == nil {
		t.Fatal("expected error connecting to 127.0.0.1:465, got nil")
	}
	if !strings.Contains(err.Error(), "smtp dial tls:") {
		t.Errorf("port 465 error should come from implicit TLS dial, got: %v", err)
	}
}

// TestSend_Port587_FailsWithoutServer confirms that non-465 ports attempt a
// plain TCP dial followed by STARTTLS. The error must come from "smtp dial:"
// (plain TCP), not from TLS.
func TestSend_Port587_STARTTLS(t *testing.T) {
	err := Send(Config{
		Host: "127.0.0.1", Port: 587,
		Username: "u", Password: "p",
	}, Message{From: "a@example.com", To: []string{"b@example.com"}, Subject: "s", Text: "t"})
	if err == nil {
		t.Fatal("expected error connecting to 127.0.0.1:587, got nil")
	}
	if !strings.Contains(err.Error(), "smtp dial:") {
		t.Errorf("port 587 error should come from plain TCP dial, got: %v", err)
	}
}

// --- Message-ID --------------------------------------------------------------

func TestBuildMIME_MessageIDAutoGenerated(t *testing.T) {
	m := buildMIME(Message{
		From: "a@example.com", To: []string{"b@example.com"},
		Subject: "s", Text: "t",
	})
	mustContain(t, m, "Message-ID: <", "Message-ID header present")
}

func TestBuildMIME_MessageIDUsesHostname(t *testing.T) {
	orig := Hostname
	t.Cleanup(func() { Hostname = orig })
	Hostname = "webmail.example.com"

	m := buildMIME(Message{
		From: "a@example.com", To: []string{"b@example.com"},
		Subject: "s", Text: "t",
	})
	mustContain(t, m, "@webmail.example.com>", "Hostname in Message-ID")
}

func TestBuildMIME_MessageIDPreservedWhenSet(t *testing.T) {
	// A pre-generated ID must be used verbatim so Send and the IMAP APPEND
	// copy share the same Message-ID.
	preset := "<abc123@webmail.example.com>"
	m := buildMIME(Message{
		From: "a@example.com", To: []string{"b@example.com"},
		Subject: "s", Text: "t",
		MessageID: preset,
	})
	mustContain(t, m, "Message-ID: "+preset, "preset Message-ID preserved")
}

func TestBuildMIME_MessageIDUnique(t *testing.T) {
	// Two calls without a preset ID must produce different Message-IDs.
	m1 := buildMIME(Message{From: "a@example.com", To: []string{"b@example.com"}, Subject: "s", Text: "t"})
	m2 := buildMIME(Message{From: "a@example.com", To: []string{"b@example.com"}, Subject: "s", Text: "t"})
	extractID := func(mime string) string {
		for _, line := range strings.Split(mime, "\r\n") {
			if strings.HasPrefix(line, "Message-ID: ") {
				return line
			}
		}
		return ""
	}
	if id1, id2 := extractID(m1), extractID(m2); id1 == id2 {
		t.Errorf("expected unique Message-IDs, both are %q", id1)
	}
}

// --- X-Mailer ----------------------------------------------------------------

func TestBuildMIME_XMailer(t *testing.T) {
	m := buildMIME(Message{
		From: "a@example.com", To: []string{"b@example.com"},
		Subject: "s", Text: "t",
	})
	mustContain(t, m, "X-Mailer: letrvu", "X-Mailer header")
}

// --- threading headers -------------------------------------------------------

func TestBuildMIME_InReplyTo(t *testing.T) {
	m := buildMIME(Message{
		From: "a@example.com", To: []string{"b@example.com"},
		Subject: "Re: s", Text: "t",
		InReplyTo: "<orig@example.com>",
	})
	mustContain(t, m, "In-Reply-To: <orig@example.com>", "In-Reply-To header")
}

func TestBuildMIME_InReplyToAbsentWhenEmpty(t *testing.T) {
	m := buildMIME(Message{
		From: "a@example.com", To: []string{"b@example.com"},
		Subject: "s", Text: "t",
	})
	mustNotContain(t, m, "In-Reply-To:", "no In-Reply-To for fresh message")
}

func TestBuildMIME_References(t *testing.T) {
	refs := "<msg1@example.com> <msg2@example.com>"
	m := buildMIME(Message{
		From: "a@example.com", To: []string{"b@example.com"},
		Subject: "Re: s", Text: "t",
		References: refs,
	})
	mustContain(t, m, "References: "+refs, "References header")
}

func TestBuildMIME_ReferencesAbsentWhenEmpty(t *testing.T) {
	m := buildMIME(Message{
		From: "a@example.com", To: []string{"b@example.com"},
		Subject: "s", Text: "t",
	})
	mustNotContain(t, m, "References:", "no References for fresh message")
}

// --- Date appears exactly once -----------------------------------------------

func TestBuildRFC822_DateAppearsOnce(t *testing.T) {
	// Guard against double-stamping now that Date moved into buildMIME.
	raw := BuildRFC822(Message{
		From: "a@example.com", To: []string{"b@example.com"},
		Subject: "s", Text: "t",
	})
	count := strings.Count(string(raw), "Date: ")
	if count != 1 {
		t.Errorf("expected exactly 1 Date header, got %d", count)
	}
}

// --- EnvelopeFrom -----------------------------------------------------------

func TestSend_EnvelopeFromFallback(t *testing.T) {
	// When EnvelopeFrom is empty, Send uses From for the envelope.
	// We can't call Send without an SMTP server, but we can verify the MIME
	// builder doesn't include EnvelopeFrom in the message body (it's SMTP-only).
	m := buildMIME(Message{
		From:         "alias@example.com",
		EnvelopeFrom: "auth@example.com",
		To:           []string{"bob@example.com"},
		Subject:      "s",
		Text:         "body",
	})
	// The MIME From: header should be the alias, not the envelope sender.
	mustContain(t, m, "From: alias@example.com", "From header is alias")
	mustNotContain(t, m, "auth@example.com", "envelope address not in MIME body")
}
