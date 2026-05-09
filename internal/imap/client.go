// Package imap wraps go-imap/v2 with higher-level operations suited to a
// webmail client. Each logged-in user gets one persistent Client that is
// reused across requests rather than reconnecting per request.
package imap

import (
	"bytes"
	"fmt"
	"io"
	"strings"
	"time"

	goimap "github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapclient"
	"github.com/emersion/go-message"
	"github.com/emersion/go-message/mail"
)

// Client wraps an authenticated IMAP connection.
type Client struct {
	c *imapclient.Client
}

// Connect dials the IMAP server over TLS and authenticates.
func Connect(host string, port int, username, password string) (*Client, error) {
	addr := fmt.Sprintf("%s:%d", host, port)
	c, err := imapclient.DialTLS(addr, nil)
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
	return folders, nil
}

// Message is a lightweight summary used in folder listings.
type Message struct {
	UID     uint32    `json:"uid"`
	Subject string    `json:"subject"`
	From    string    `json:"from"`
	Date    time.Time `json:"date"`
	Read    bool      `json:"read"`
	Size    uint32    `json:"size"`
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
		UID:        true,
		Envelope:   true,
		Flags:      true,
		RFC822Size: true,
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
			if flag == goimap.FlagSeen {
				msg.Read = true
				break
			}
		}
		result = append(result, msg)
	}
	return result, nil
}

// MessageFull holds the complete content of a single message.
type MessageFull struct {
	UID         uint32    `json:"uid"`
	Subject     string    `json:"subject"`
	From        string    `json:"from"`
	To          []string  `json:"to"`
	CC          []string  `json:"cc"`
	Date        time.Time `json:"date"`
	TextBody    string    `json:"text_body"`
	HTMLBody    string    `json:"html_body"`
	Attachments []string  `json:"attachments"`
}

// GetMessage fetches the full content of a single message by UID.
func (c *Client) GetMessage(folder string, uid uint32) (*MessageFull, error) {
	if _, err := c.c.Select(folder, nil).Wait(); err != nil {
		return nil, fmt.Errorf("select %q: %w", folder, err)
	}

	uidSet := goimap.UIDSetNum(goimap.UID(uid))
	fetchOpts := &goimap.FetchOptions{
		UID:      true,
		Envelope: true,
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
			}
		case *mail.AttachmentHeader:
			filename, _ := h.Filename()
			full.Attachments = append(full.Attachments, filename)
		}
	}
	return nil
}

// DeleteMessage moves a message to Trash by UID.
func (c *Client) DeleteMessage(folder string, uid uint32) error {
	// TODO: add \Deleted flag + EXPUNGE, or MOVE to Trash
	return fmt.Errorf("DeleteMessage: not implemented")
}

// MarkRead sets or clears the \Seen flag on a message.
func (c *Client) MarkRead(folder string, uid uint32, read bool) error {
	// TODO: c.c.UIDStore() with +FLAGS or -FLAGS \Seen
	return fmt.Errorf("MarkRead: not implemented")
}

func formatAddress(addr goimap.Address) string {
	if addr.Name != "" {
		return addr.Name + " <" + addr.Addr() + ">"
	}
	return addr.Addr()
}
