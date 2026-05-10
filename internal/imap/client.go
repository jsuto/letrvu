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
	"mime"
	"slices"
	"strings"
	"time"

	goimap "github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapclient"
	"github.com/emersion/go-message"
	"github.com/emersion/go-message/charset"
	"github.com/emersion/go-message/mail"
)

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
	Name      string `json:"name"`
	Delimiter string `json:"delimiter"`
}

// ListFolders returns all subscribed mailbox folders.
func (c *Client) ListFolders() ([]Folder, error) {
	mailboxes, err := c.c.List("", "*", nil).Collect()
	if err != nil {
		return nil, fmt.Errorf("list folders: %w", err)
	}
	folders := make([]Folder, len(mailboxes))
	for i, data := range mailboxes {
		delim := ""
		if data.Delim != 0 {
			delim = string(data.Delim)
		}
		folders[i] = Folder{
			Name:      data.Mailbox,
			Delimiter: delim,
		}
	}
	slices.SortFunc(folders, func(a, b Folder) int {
		return strings.Compare(strings.ToLower(a.Name), strings.ToLower(b.Name))
	})
	return folders, nil
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
		if len(buf.Envelope.From) > 0 {
			full.From = formatAddress(buf.Envelope.From[0])
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

// parseMIMEBody walks the MIME tree and populates text/HTML bodies and
// attachment filenames on full.
func parseMIMEBody(raw []byte, full *MessageFull) error {
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
