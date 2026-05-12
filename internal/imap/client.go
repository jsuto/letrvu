// Package imap wraps go-imap/v2 with higher-level operations suited to a
// webmail client. Each logged-in user gets one persistent Client that is
// reused across requests rather than reconnecting per request.
package imap

import (
	"bytes"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"mime"
	"regexp"
	"slices"
	"strings"
	"time"

	goimap "github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapclient"
	"github.com/emersion/go-message"
	"github.com/emersion/go-message/charset"
	"github.com/emersion/go-message/mail"
)

// Debug enables verbose logging for development. Set via LOG_LEVEL=debug.
var Debug bool

func debugf(format string, args ...any) {
	if Debug {
		log.Printf("[imap debug] "+format, args...)
	}
}

// DefaultTLSConfig is used for every new IMAP connection. main() may replace
// it before the server starts (e.g. to toggle InsecureSkipVerify).
var DefaultTLSConfig = &tls.Config{
	InsecureSkipVerify: true, //nolint:gosec
}

// Client wraps an authenticated IMAP connection.
type Client struct {
	c *imapclient.Client
}

// dialOptions returns imapclient.Options with TLS and RFC 2047 word decoding
// configured. Callers may extend the returned value before dialling.
func dialOptions(host string) *imapclient.Options {
	tlsCfg := DefaultTLSConfig.Clone()
	tlsCfg.ServerName = host
	return &imapclient.Options{
		TLSConfig:   tlsCfg,
		WordDecoder: &mime.WordDecoder{CharsetReader: charset.Reader},
	}
}

// Connect dials the IMAP server over TLS and authenticates.
// TLS behaviour is controlled by DefaultTLSConfig.
func Connect(host string, port int, username, password string) (*Client, error) {
	addr := fmt.Sprintf("%s:%d", host, port)
	c, err := imapclient.DialTLS(addr, dialOptions(host))
	if err != nil {
		return nil, fmt.Errorf("imap dial %s: %w", addr, err)
	}
	if err := c.Login(username, password).Wait(); err != nil {
		c.Close()
		return nil, fmt.Errorf("imap login: %w", err)
	}
	return &Client{c: c}, nil
}

// Close logs out and closes the connection gracefully.
func (c *Client) Close() error {
	return c.c.Logout().Wait()
}

// Folder is a mailbox folder.
type Folder struct {
	Name       string `json:"name"`
	Delimiter  string `json:"delimiter"`
	Unseen     uint32 `json:"unseen"`
	Subscribed bool   `json:"subscribed"`
}

// folderPriority returns a sort key so well-known folders appear first.
// INBOX=0, Drafts=1, Sent=2, Junk/Spam=3, Trash=4, everything else=5.
func folderPriority(name string) int {
	switch strings.ToLower(name) {
	case "inbox":
		return 0
	case "drafts", "draft":
		return 1
	case "sent", "sent items", "sent mail":
		return 2
	case "junk", "junk email", "spam":
		return 3
	case "trash", "deleted", "deleted items":
		return 4
	}
	return 5
}

// ListFolders returns all mailbox folders with unseen counts and subscription
// status, sorted by well-known order then alphabetically. The Subscribed field
// is true when the server reports the \Subscribed attribute or when it is in
// the subscribed set (LIST RETURN (SUBSCRIBED)).
func (c *Client) ListFolders() ([]Folder, error) {
	mailboxes, err := c.c.List("", "*", &goimap.ListOptions{ReturnSubscribed: true}).Collect()
	if err != nil {
		// Retry without the RETURN extension in case the server doesn't support it.
		mailboxes, err = c.c.List("", "*", nil).Collect()
		if err != nil {
			return nil, fmt.Errorf("list folders: %w", err)
		}
	}
	folders := make([]Folder, 0, len(mailboxes))
	for _, data := range mailboxes {
		delim := ""
		if data.Delim != 0 {
			delim = string(data.Delim)
		}
		subscribed := false
		for _, attr := range data.Attrs {
			if strings.EqualFold(string(attr), string(goimap.MailboxAttrSubscribed)) {
				subscribed = true
				break
			}
		}
		f := Folder{
			Name:       data.Mailbox,
			Delimiter:  delim,
			Subscribed: subscribed,
		}
		// Fetch unseen count via STATUS; ignore errors (best-effort).
		if status, err := c.c.Status(data.Mailbox, &goimap.StatusOptions{NumUnseen: true}).Wait(); err == nil && status.NumUnseen != nil {
			f.Unseen = *status.NumUnseen
		}
		folders = append(folders, f)
	}
	slices.SortFunc(folders, func(a, b Folder) int {
		pa, pb := folderPriority(a.Name), folderPriority(b.Name)
		if pa != pb {
			return pa - pb
		}
		return strings.Compare(strings.ToLower(a.Name), strings.ToLower(b.Name))
	})
	return folders, nil
}

// Subscribe adds the named mailbox to the user's subscription list.
func (c *Client) Subscribe(folder string) error {
	if err := c.c.Subscribe(folder).Wait(); err != nil {
		return fmt.Errorf("subscribe %q: %w", folder, err)
	}
	return nil
}

// Unsubscribe removes the named mailbox from the user's subscription list.
func (c *Client) Unsubscribe(folder string) error {
	if err := c.c.Unsubscribe(folder).Wait(); err != nil {
		return fmt.Errorf("unsubscribe %q: %w", folder, err)
	}
	return nil
}

// CreateFolder creates a new mailbox with the given name.
func (c *Client) CreateFolder(name string) error {
	if err := c.c.Create(name, nil).Wait(); err != nil {
		return fmt.Errorf("create folder %q: %w", name, err)
	}
	return nil
}

// RenameFolder renames an existing mailbox.
func (c *Client) RenameFolder(oldName, newName string) error {
	if err := c.c.Rename(oldName, newName).Wait(); err != nil {
		return fmt.Errorf("rename folder %q → %q: %w", oldName, newName, err)
	}
	return nil
}

// DeleteFolder permanently removes a mailbox and all its messages.
func (c *Client) DeleteFolder(name string) error {
	if err := c.c.Delete(name).Wait(); err != nil {
		return fmt.Errorf("delete folder %q: %w", name, err)
	}
	return nil
}

// Message is a lightweight summary used in folder listings.
type Message struct {
	UID            uint32    `json:"uid"`
	Subject        string    `json:"subject"`
	From           string    `json:"from"`
	Date           time.Time `json:"date"`
	Read           bool      `json:"read"`
	Flagged        bool      `json:"flagged"`
	HasAttachments bool      `json:"has_attachments"`
	Size           uint32    `json:"size"`
	// Threading headers — used by the frontend to group messages into threads.
	MessageID  string `json:"message_id,omitempty"`
	InReplyTo  string `json:"in_reply_to,omitempty"`
	References string `json:"references,omitempty"`
}

// ListMessages returns message summaries for a folder, newest first.
// page is 1-indexed; pageSize controls how many messages per page.
func (c *Client) ListMessages(folder string, page, pageSize int) ([]Message, error) {
	selectData, err := c.c.Select(folder, nil).Wait()
	if err != nil {
		return nil, fmt.Errorf("select %q: %w", folder, err)
	}

	total := int(selectData.NumMessages)
	if total == 0 {
		return []Message{}, nil
	}

	// Newest first: highest sequence numbers are newest messages.
	end := total - (page-1)*pageSize
	if end <= 0 {
		return []Message{}, nil
	}
	start := end - pageSize + 1
	if start < 1 {
		start = 1
	}

	var seqSet goimap.SeqSet
	seqSet.AddRange(uint32(start), uint32(end))

	fetchOpts := &goimap.FetchOptions{
		UID:           true,
		Envelope:      true,
		Flags:         true,
		RFC822Size:    true,
		BodyStructure: &goimap.FetchItemBodyStructure{Extended: true},
	}

	msgs, err := c.c.Fetch(seqSet, fetchOpts).Collect()
	if err != nil {
		return nil, fmt.Errorf("fetch messages: %w", err)
	}

	// Reverse so the highest sequence number (newest) is first.
	result := make([]Message, 0, len(msgs))
	for i := len(msgs) - 1; i >= 0; i-- {
		buf := msgs[i]
		msg := Message{
			UID:  uint32(buf.UID),
			Size: uint32(buf.RFC822Size),
		}
		if buf.Envelope != nil {
			msg.Subject = buf.Envelope.Subject
			msg.Date = buf.Envelope.Date
			msg.MessageID = buf.Envelope.MessageID
			if len(buf.Envelope.From) > 0 {
				msg.From = formatAddress(buf.Envelope.From[0])
			}
		}
		for _, flag := range buf.Flags {
			switch flag {
			case goimap.FlagSeen:
				msg.Read = true
			case goimap.FlagFlagged:
				msg.Flagged = true
			}
		}
		if buf.BodyStructure != nil {
			msg.HasAttachments = bodyHasAttachments(buf.BodyStructure)
		}
		result = append(result, msg)
	}

	// Second pass: fetch threading headers (In-Reply-To, References) for the
	// same sequence set. Done separately so a failure here does not affect the
	// message listing — threading is best-effort.
	c.enrichWithThreadHeaders(result, seqSet)

	return result, nil
}

// enrichWithThreadHeaders fetches In-Reply-To and References headers for the
// given messages in a single FETCH and populates the fields in-place.
// Failures are silently ignored so the caller always gets the base list.
func (c *Client) enrichWithThreadHeaders(msgs []Message, seqSet goimap.SeqSet) {
	// Build a UID→index map for fast lookup.
	byUID := make(map[uint32]int, len(msgs))
	for i, m := range msgs {
		byUID[m.UID] = i
	}

	threadFetch := &goimap.FetchOptions{
		UID: true,
		BodySection: []*goimap.FetchItemBodySection{
			{
				Peek:         true,
				Specifier:    goimap.PartSpecifierHeader,
				HeaderFields: []string{"MESSAGE-ID", "IN-REPLY-TO", "REFERENCES"},
			},
		},
	}
	fetched, err := c.c.Fetch(seqSet, threadFetch).Collect()
	if err != nil {
		debugf("enrichWithThreadHeaders: fetch failed (non-fatal): %v", err)
		return
	}
	for _, buf := range fetched {
		idx, ok := byUID[uint32(buf.UID)]
		if !ok {
			continue
		}
		for _, bodyBytes := range buf.BodySection {
			rh := rawHeaders(bodyBytes)
			if vals := rh["message-id"]; len(vals) > 0 {
				msgs[idx].MessageID = strings.TrimSpace(vals[0])
			}
			if vals := rh["in-reply-to"]; len(vals) > 0 {
				msgs[idx].InReplyTo = strings.TrimSpace(vals[0])
			}
			if vals := rh["references"]; len(vals) > 0 {
				msgs[idx].References = strings.TrimSpace(vals[0])
			}
		}
	}
}

// Attachment describes a message attachment.
type Attachment struct {
	Index       int    `json:"index"`
	Filename    string `json:"filename"`
	ContentType string `json:"content_type"`
	Size        int    `json:"size"`
}

// MessageFull holds the complete content of a single message.
type MessageFull struct {
	UID            uint32            `json:"uid"`
	Subject        string            `json:"subject"`
	From           string            `json:"from"`
	ReplyTo        string            `json:"reply_to,omitempty"`
	To             []string          `json:"to"`
	CC             []string          `json:"cc"`
	Date           time.Time         `json:"date"`
	Read           bool              `json:"read"`
	Flagged        bool              `json:"flagged"`
	HasAttachments bool              `json:"has_attachments"`
	TextBody       string            `json:"text_body"`
	HTMLBody       string            `json:"html_body"`
	Attachments    []Attachment      `json:"attachments"`
	ICalInvite     string            `json:"ical_invite,omitempty"`
	// InlineImages maps Content-ID (without angle brackets) to a base64 data
	// URL so the frontend can resolve cid: references inside HTML bodies.
	InlineImages   map[string]string `json:"inline_images,omitempty"`
	// SPF/DKIM/DMARC results parsed from the Authentication-Results header
	// added by the receiving MTA. Values are lowercase result tokens such as
	// "pass", "fail", "softfail", "neutral", "none". Empty when absent.
	SPF   string `json:"spf,omitempty"`
	DKIM  string `json:"dkim,omitempty"`
	DMARC string `json:"dmarc,omitempty"`

	// Threading headers — needed by the frontend to set In-Reply-To /
	// References when the user replies to this message.
	MessageID  string `json:"message_id,omitempty"`
	References string `json:"references,omitempty"`
}

// GetMessage fetches the full content of a single message by UID.
func (c *Client) GetMessage(folder string, uid uint32) (*MessageFull, error) {
	if _, err := c.c.Select(folder, nil).Wait(); err != nil {
		return nil, fmt.Errorf("select %q: %w", folder, err)
	}

	uidSet := goimap.UIDSetNum(goimap.UID(uid))
	fetchOpts := &goimap.FetchOptions{
		UID:           true,
		Envelope:      true,
		Flags:         true,
		BodyStructure: &goimap.FetchItemBodyStructure{Extended: true},
		BodySection: []*goimap.FetchItemBodySection{
			{Peek: true}, // BODY.PEEK[] — fetch entire message without marking Seen
		},
	}

	msgs, err := c.c.Fetch(uidSet, fetchOpts).Collect()
	if err != nil {
		return nil, fmt.Errorf("uid fetch: %w", err)
	}
	if len(msgs) == 0 {
		return nil, fmt.Errorf("message uid %d not found in %q", uid, folder)
	}

	buf := msgs[0]
	full := &MessageFull{
		UID: uint32(buf.UID),
		To:  []string{},
		CC:  []string{},
	}

	if buf.Envelope != nil {
		full.Subject = buf.Envelope.Subject
		full.Date = buf.Envelope.Date
		full.MessageID = buf.Envelope.MessageID
		if len(buf.Envelope.From) > 0 {
			full.From = formatAddress(buf.Envelope.From[0])
		}
		if len(buf.Envelope.ReplyTo) > 0 {
			full.ReplyTo = formatAddress(buf.Envelope.ReplyTo[0])
		}
		for _, addr := range buf.Envelope.To {
			if a := addr.Addr(); a != "" {
				full.To = append(full.To, a)
			}
		}
		for _, addr := range buf.Envelope.Cc {
			if a := addr.Addr(); a != "" {
				full.CC = append(full.CC, a)
			}
		}
	}
	for _, flag := range buf.Flags {
		switch flag {
		case goimap.FlagSeen:
			full.Read = true
		case goimap.FlagFlagged:
			full.Flagged = true
		}
	}
	if buf.BodyStructure != nil {
		full.HasAttachments = bodyHasAttachments(buf.BodyStructure)
	}

	// Parse MIME body from the fetched literal.
	for _, bodyBytes := range buf.BodySection {
		if err := parseMIMEBody(bodyBytes, full); err != nil {
			return nil, fmt.Errorf("parse mime: %w", err)
		}
	}

	return full, nil
}

// authMethodRe matches "method=result" tokens in an Authentication-Results
// header value, e.g. "spf=pass", "dkim=fail", "dmarc=none".
var authMethodRe = regexp.MustCompile(`(?i)\b(spf|dkim|dmarc)\s*=\s*(\w+)`)

// parseAuthResults extracts the first SPF, DKIM, and DMARC result tokens from
// an Authentication-Results header value. Results are lower-cased.
func parseAuthResults(v string) (spf, dkim, dmarc string) {
	for _, m := range authMethodRe.FindAllStringSubmatch(v, -1) {
		switch strings.ToLower(m[1]) {
		case "spf":
			if spf == "" {
				spf = strings.ToLower(m[2])
			}
		case "dkim":
			if dkim == "" {
				dkim = strings.ToLower(m[2])
			}
		case "dmarc":
			if dmarc == "" {
				dmarc = strings.ToLower(m[2])
			}
		}
	}
	return
}

// headerKeys returns the sorted keys of a raw-headers map for debug logging.
func headerKeys(h map[string][]string) []string {
	keys := make([]string, 0, len(h))
	for k := range h {
		keys = append(keys, k)
	}
	return keys
}

// rawHeaders parses the header section of a raw RFC 2822 message (everything
// before the first blank line) into a map of lowercased field names to their
// unfolded values. Multi-line (folded) header values are joined with a space.
// Multiple fields with the same name are appended in order.
func rawHeaders(msg []byte) map[string][]string {
	out := make(map[string][]string)
	var key string
	var val strings.Builder

	flush := func() {
		if key != "" {
			out[key] = append(out[key], strings.TrimSpace(val.String()))
		}
	}

	for len(msg) > 0 {
		nl := bytes.IndexByte(msg, '\n')
		var line []byte
		if nl < 0 {
			line = msg
			msg = nil
		} else {
			line = msg[:nl]
			msg = msg[nl+1:]
		}
		line = bytes.TrimRight(line, "\r") // strip CRLF → LF

		if len(line) == 0 {
			break // blank line = end of headers
		}
		if line[0] == ' ' || line[0] == '\t' {
			// Continuation of previous field value.
			val.WriteByte(' ')
			val.Write(bytes.TrimLeft(line, " \t"))
			continue
		}
		flush()
		key = ""
		val.Reset()
		if colon := bytes.IndexByte(line, ':'); colon > 0 {
			key = strings.ToLower(string(line[:colon]))
			val.Write(bytes.TrimLeft(line[colon+1:], " \t"))
		}
	}
	flush()
	return out
}

// parseMIMEBody walks the MIME tree and populates text/HTML bodies and
// attachment filenames on full.
func parseMIMEBody(raw []byte, full *MessageFull) error {
	// Extract email authentication results directly from the raw header bytes.
	// We do this before calling mail.CreateReader to avoid any ambiguity in
	// how go-message handles folded header values. The headers are written by
	// the receiving MTA and cannot be spoofed by the sender.
	rh := rawHeaders(raw)
	debugf("uid=%d raw header keys: %v", full.UID, headerKeys(rh))

	// References header (RFC 2822 threading).
	if refs := rh["references"]; len(refs) > 0 {
		full.References = strings.TrimSpace(refs[0])
	}

	for _, v := range rh["authentication-results"] {
		debugf("uid=%d Authentication-Results value: %q", full.UID, v)
		spf, dkim, dmarc := parseAuthResults(v)
		debugf("uid=%d parsed → spf=%q dkim=%q dmarc=%q", full.UID, spf, dkim, dmarc)
		if full.SPF == "" && spf != "" {
			full.SPF = spf
		}
		if full.DKIM == "" && dkim != "" {
			full.DKIM = dkim
		}
		if full.DMARC == "" && dmarc != "" {
			full.DMARC = dmarc
		}
	}
	// Received-SPF fallback (used by some MTAs). Format: "Pass (...)" / "Fail ..."
	if full.SPF == "" {
		for _, v := range rh["received-spf"] {
			debugf("uid=%d Received-SPF value: %q", full.UID, v)
			if parts := strings.Fields(strings.ToLower(v)); len(parts) > 0 {
				full.SPF = strings.TrimRight(parts[0], ":")
				break
			}
		}
	}
	debugf("uid=%d final auth: spf=%q dkim=%q dmarc=%q", full.UID, full.SPF, full.DKIM, full.DMARC)

	mr, err := mail.CreateReader(bytes.NewReader(raw))
	if err != nil && !message.IsUnknownCharset(err) {
		return err
	}

	for {
		part, err := mr.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil && !message.IsUnknownCharset(err) {
			break
		}
		switch h := part.Header.(type) {
		case *mail.InlineHeader:
			ct, _, _ := h.ContentType()
			body, _ := io.ReadAll(part.Body)
			switch {
			case strings.HasPrefix(ct, "text/html"):
				full.HTMLBody = string(body)
			case strings.HasPrefix(ct, "text/plain"):
				full.TextBody = string(body)
			case strings.HasPrefix(ct, "text/calendar"):
				full.ICalInvite = string(body)
			default:
				// Inline image or other binary part referenced by cid: in HTML.
				cid := strings.Trim(h.Get("Content-Id"), "<>")
				if cid != "" && len(body) > 0 {
					if full.InlineImages == nil {
						full.InlineImages = make(map[string]string)
					}
					full.InlineImages[cid] = "data:" + ct + ";base64," + base64.StdEncoding.EncodeToString(body)
				}
			}
		case *mail.AttachmentHeader:
			body, _ := io.ReadAll(part.Body)
			ct, _, _ := h.ContentType()
			filename, _ := h.Filename()
			full.Attachments = append(full.Attachments, Attachment{
				Index:       len(full.Attachments),
				Filename:    filename,
				ContentType: ct,
				Size:        len(body),
			})
		}
	}
	return nil
}

// GetRawMessage returns the complete RFC 2822 source of a message (headers +
// body) without marking it as Seen.
func (c *Client) GetRawMessage(folder string, uid uint32) ([]byte, error) {
	if _, err := c.c.Select(folder, nil).Wait(); err != nil {
		return nil, fmt.Errorf("select %q: %w", folder, err)
	}
	uidSet := goimap.UIDSetNum(goimap.UID(uid))
	fetchOpts := &goimap.FetchOptions{
		BodySection: []*goimap.FetchItemBodySection{{Peek: true}},
	}
	msgs, err := c.c.Fetch(uidSet, fetchOpts).Collect()
	if err != nil {
		return nil, fmt.Errorf("uid fetch: %w", err)
	}
	if len(msgs) == 0 {
		return nil, fmt.Errorf("message uid %d not found in %q", uid, folder)
	}
	for _, b := range msgs[0].BodySection {
		return b, nil
	}
	return nil, fmt.Errorf("no body section returned for uid %d", uid)
}

// DownloadAttachment fetches the raw bytes of the attachment at index i.
func (c *Client) DownloadAttachment(folder string, uid uint32, index int) (*Attachment, []byte, error) {
	if _, err := c.c.Select(folder, nil).Wait(); err != nil {
		return nil, nil, fmt.Errorf("select %q: %w", folder, err)
	}

	uidSet := goimap.UIDSetNum(goimap.UID(uid))
	fetchOpts := &goimap.FetchOptions{
		BodySection: []*goimap.FetchItemBodySection{{Peek: true}},
	}

	msgs, err := c.c.Fetch(uidSet, fetchOpts).Collect()
	if err != nil {
		return nil, nil, fmt.Errorf("uid fetch: %w", err)
	}
	if len(msgs) == 0 {
		return nil, nil, fmt.Errorf("message uid %d not found", uid)
	}

	for _, bodyBytes := range msgs[0].BodySection {
		mr, err := mail.CreateReader(bytes.NewReader(bodyBytes))
		if err != nil && !message.IsUnknownCharset(err) {
			return nil, nil, err
		}
		i := 0
		for {
			part, err := mr.NextPart()
			if err == io.EOF {
				break
			}
			if err != nil && !message.IsUnknownCharset(err) {
				break
			}
			ah, ok := part.Header.(*mail.AttachmentHeader)
			if !ok {
				continue
			}
			if i == index {
				data, _ := io.ReadAll(part.Body)
				ct, _, _ := ah.ContentType()
				filename, _ := ah.Filename()
				return &Attachment{
					Index:       i,
					Filename:    filename,
					ContentType: ct,
					Size:        len(data),
				}, data, nil
			}
			io.Copy(io.Discard, part.Body) //nolint:errcheck
			i++
		}
	}
	return nil, nil, fmt.Errorf("attachment index %d not found", index)
}

// SearchMessages runs a server-side TEXT search in folder and returns matching
// message summaries, newest first.
func (c *Client) SearchMessages(folder, query string) ([]Message, error) {
	if _, err := c.c.Select(folder, nil).Wait(); err != nil {
		return nil, fmt.Errorf("select %q: %w", folder, err)
	}

	criteria := &goimap.SearchCriteria{
		Text: []string{query},
	}
	searchData, err := c.c.UIDSearch(criteria, nil).Wait()
	if err != nil {
		return nil, fmt.Errorf("uid search: %w", err)
	}

	uids := searchData.AllUIDs()
	if len(uids) == 0 {
		return []Message{}, nil
	}

	var uidSet goimap.UIDSet
	for _, uid := range uids {
		uidSet.AddNum(uid)
	}

	fetchOpts := &goimap.FetchOptions{
		UID:           true,
		Envelope:      true,
		Flags:         true,
		RFC822Size:    true,
		BodyStructure: &goimap.FetchItemBodyStructure{Extended: true},
	}
	msgs, err := c.c.Fetch(uidSet, fetchOpts).Collect()
	if err != nil {
		return nil, fmt.Errorf("fetch: %w", err)
	}

	result := make([]Message, 0, len(msgs))
	for i := len(msgs) - 1; i >= 0; i-- {
		buf := msgs[i]
		msg := Message{
			UID:  uint32(buf.UID),
			Size: uint32(buf.RFC822Size),
		}
		if buf.Envelope != nil {
			msg.Subject = buf.Envelope.Subject
			msg.Date = buf.Envelope.Date
			if len(buf.Envelope.From) > 0 {
				msg.From = formatAddress(buf.Envelope.From[0])
			}
		}
		for _, flag := range buf.Flags {
			switch flag {
			case goimap.FlagSeen:
				msg.Read = true
			case goimap.FlagFlagged:
				msg.Flagged = true
			}
		}
		if buf.BodyStructure != nil {
			msg.HasAttachments = bodyHasAttachments(buf.BodyStructure)
		}
		result = append(result, msg)
	}
	return result, nil
}

// DeleteMessage sets \Deleted on the message and expunges it.
func (c *Client) DeleteMessage(folder string, uid uint32) error {
	if _, err := c.c.Select(folder, nil).Wait(); err != nil {
		return fmt.Errorf("select %q: %w", folder, err)
	}
	uidSet := goimap.UIDSetNum(goimap.UID(uid))
	if err := c.c.Store(uidSet, &goimap.StoreFlags{
		Op:     goimap.StoreFlagsAdd,
		Silent: true,
		Flags:  []goimap.Flag{goimap.FlagDeleted},
	}, nil).Close(); err != nil {
		return fmt.Errorf("mark deleted: %w", err)
	}
	return c.c.Expunge().Close()
}

// DeleteMessages sets \Deleted on all given UIDs and expunges them in one pass.
func (c *Client) DeleteMessages(folder string, uids []uint32) error {
	if _, err := c.c.Select(folder, nil).Wait(); err != nil {
		return fmt.Errorf("select %q: %w", folder, err)
	}
	var uidSet goimap.UIDSet
	for _, uid := range uids {
		uidSet.AddNum(goimap.UID(uid))
	}
	if err := c.c.Store(uidSet, &goimap.StoreFlags{
		Op:     goimap.StoreFlagsAdd,
		Silent: true,
		Flags:  []goimap.Flag{goimap.FlagDeleted},
	}, nil).Close(); err != nil {
		return fmt.Errorf("mark deleted: %w", err)
	}
	return c.c.Expunge().Close()
}

// MarkReadMessages sets or clears \Seen on multiple UIDs in one STORE command.
func (c *Client) MarkReadMessages(folder string, uids []uint32, read bool) error {
	if _, err := c.c.Select(folder, nil).Wait(); err != nil {
		return fmt.Errorf("select %q: %w", folder, err)
	}
	var uidSet goimap.UIDSet
	for _, uid := range uids {
		uidSet.AddNum(goimap.UID(uid))
	}
	op := goimap.StoreFlagsAdd
	if !read {
		op = goimap.StoreFlagsDel
	}
	return c.c.Store(uidSet, &goimap.StoreFlags{
		Op:     op,
		Silent: true,
		Flags:  []goimap.Flag{goimap.FlagSeen},
	}, nil).Close()
}

// MoveMessage moves a single message to another folder using IMAP MOVE (RFC 6851).
func (c *Client) MoveMessage(folder string, uid uint32, destFolder string) error {
	return c.MoveMessages(folder, []uint32{uid}, destFolder)
}

// MoveMessages moves multiple messages to another folder in one IMAP MOVE command.
func (c *Client) MoveMessages(folder string, uids []uint32, destFolder string) error {
	if _, err := c.c.Select(folder, nil).Wait(); err != nil {
		return fmt.Errorf("select %q: %w", folder, err)
	}
	var uidSet goimap.UIDSet
	for _, uid := range uids {
		uidSet.AddNum(goimap.UID(uid))
	}
	_, err := c.c.Move(uidSet, destFolder).Wait()
	return err
}

// MarkFlagged sets or clears the \Flagged flag on a message.
func (c *Client) MarkFlagged(folder string, uid uint32, flagged bool) error {
	if _, err := c.c.Select(folder, nil).Wait(); err != nil {
		return fmt.Errorf("select %q: %w", folder, err)
	}
	op := goimap.StoreFlagsAdd
	if !flagged {
		op = goimap.StoreFlagsDel
	}
	return c.c.Store(goimap.UIDSetNum(goimap.UID(uid)), &goimap.StoreFlags{
		Op:     op,
		Silent: true,
		Flags:  []goimap.Flag{goimap.FlagFlagged},
	}, nil).Close()
}

// MarkRead sets or clears the \Seen flag on a message.
func (c *Client) MarkRead(folder string, uid uint32, read bool) error {
	if _, err := c.c.Select(folder, nil).Wait(); err != nil {
		return fmt.Errorf("select %q: %w", folder, err)
	}
	op := goimap.StoreFlagsAdd
	if !read {
		op = goimap.StoreFlagsDel
	}
	return c.c.Store(goimap.UIDSetNum(goimap.UID(uid)), &goimap.StoreFlags{
		Op:     op,
		Silent: true,
		Flags:  []goimap.Flag{goimap.FlagSeen},
	}, nil).Close()
}

// SaveDraft appends a raw RFC 2822 message to the Drafts mailbox with the
// \Draft and \Seen flags. The folder is discovered by special-use attribute
// first, then by common name patterns; falls back to "Drafts".
func (c *Client) SaveDraft(msg []byte) error {
	folder := c.findSpecialFolder(goimap.MailboxAttrDrafts, []string{"drafts", "draft"})
	if folder == "" {
		folder = "Drafts"
	}
	return c.appendMessage(folder, []goimap.Flag{goimap.FlagDraft, goimap.FlagSeen}, msg)
}

// JunkFolder returns the name of the Junk/Spam mailbox. It checks for the
// \Junk special-use attribute first, then falls back to common name patterns,
// and finally returns "Junk" if nothing else matches.
func (c *Client) JunkFolder() string {
	folder := c.findSpecialFolder(goimap.MailboxAttrJunk, []string{"junk", "junk email", "spam"})
	if folder == "" {
		folder = "Junk"
	}
	return folder
}

// SaveSent appends a raw RFC 2822 message to the Sent mailbox with the \Seen
// flag. The folder is discovered by special-use attribute first, then by
// common name patterns; falls back to "Sent".
func (c *Client) SaveSent(msg []byte) error {
	folder := c.findSpecialFolder(goimap.MailboxAttrSent, []string{"sent", "sent items", "sent mail"})
	if folder == "" {
		folder = "Sent"
	}
	return c.appendMessage(folder, []goimap.Flag{goimap.FlagSeen}, msg)
}

// appendMessage appends a raw RFC 2822 message to the named mailbox with the
// given flags set on arrival.
func (c *Client) appendMessage(folder string, flags []goimap.Flag, msg []byte) error {
	opts := &goimap.AppendOptions{Flags: flags, Time: time.Now()}
	cmd := c.c.Append(folder, int64(len(msg)), opts)
	if _, err := cmd.Write(msg); err != nil {
		return fmt.Errorf("append write: %w", err)
	}
	if err := cmd.Close(); err != nil {
		return fmt.Errorf("append close: %w", err)
	}
	if _, err := cmd.Wait(); err != nil {
		return fmt.Errorf("append: %w", err)
	}
	return nil
}

// findSpecialFolder returns the name of the mailbox that carries the given
// special-use attribute (e.g. \Drafts, \Sent). If no attribute match is found,
// it falls back to the first mailbox whose lowercased name appears in
// fallbackNames. Returns "" when nothing matches.
func (c *Client) findSpecialFolder(attr goimap.MailboxAttr, fallbackNames []string) string {
	mailboxes, err := c.c.List("", "*", nil).Collect()
	if err != nil {
		return ""
	}
	return pickSpecialFolder(mailboxes, attr, fallbackNames)
}

// pickSpecialFolder is the pure, testable core of findSpecialFolder.
func pickSpecialFolder(mailboxes []*goimap.ListData, attr goimap.MailboxAttr, fallbackNames []string) string {
	// First pass: special-use attribute reported by the server.
	for _, mb := range mailboxes {
		for _, a := range mb.Attrs {
			if strings.EqualFold(string(a), string(attr)) {
				return mb.Mailbox
			}
		}
	}
	// Second pass: well-known name patterns.
	nameSet := make(map[string]struct{}, len(fallbackNames))
	for _, n := range fallbackNames {
		nameSet[n] = struct{}{}
	}
	for _, mb := range mailboxes {
		if _, ok := nameSet[strings.ToLower(mb.Mailbox)]; ok {
			return mb.Mailbox
		}
	}
	return ""
}

// bodyHasAttachments walks the body structure and returns true if any part
// has a Content-Disposition of "attachment".
func bodyHasAttachments(bs goimap.BodyStructure) bool {
	found := false
	bs.Walk(func(path []int, part goimap.BodyStructure) bool {
		if found {
			return false
		}
		d := part.Disposition()
		if d != nil && strings.EqualFold(d.Value, "attachment") {
			found = true
		}
		return true
	})
	return found
}

func formatAddress(addr goimap.Address) string {
	if addr.Name != "" {
		return addr.Name + " <" + addr.Addr() + ">"
	}
	return addr.Addr()
}
