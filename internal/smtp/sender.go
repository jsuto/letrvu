// Package smtp handles outbound mail delivery.
package smtp

import (
	"encoding/base64"
	"fmt"
	"net/smtp"
	"strings"
	"time"
)

// Config holds the SMTP server connection details.
type Config struct {
	Host     string
	Port     int
	Username string
	Password string
}

// Attachment is a file to be attached to an outbound message.
type Attachment struct {
	Filename    string
	ContentType string // defaults to application/octet-stream if empty
	Data        []byte
}

// Message is an outbound email.
type Message struct {
	// From is the RFC 5322 From: header value (what recipients see).
	// Format: "Name <email>" or plain "email".
	From string
	// EnvelopeFrom is the SMTP MAIL FROM: address used for bounce routing.
	// If empty, From is used. Set this to the authenticated SMTP username
	// when sending from an alias so the server does not reject the command.
	EnvelopeFrom string
	To           []string
	CC           []string
	Subject      string
	Text         string       // plain text body
	HTML         string       // optional HTML body; if set, sends multipart/alternative
	Attachments  []Attachment // optional file attachments
}

// Send delivers msg via SMTP STARTTLS with PLAIN auth.
func Send(cfg Config, msg Message) error {
	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	auth := smtp.PlainAuth("", cfg.Username, cfg.Password, cfg.Host)

	envelopeFrom := msg.EnvelopeFrom
	if envelopeFrom == "" {
		envelopeFrom = msg.From
	}

	recipients := append(msg.To, msg.CC...)
	body := buildMIME(msg)

	if err := smtp.SendMail(addr, auth, envelopeFrom, recipients, []byte(body)); err != nil {
		return fmt.Errorf("smtp send: %w", err)
	}
	return nil
}

// BuildRFC822 returns the complete RFC 2822 message bytes, including a Date
// header. It is used to save drafts via IMAP APPEND.
func BuildRFC822(msg Message) []byte {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Date: %s\r\n", time.Now().Format(time.RFC1123Z)))
	sb.WriteString(buildMIME(msg))
	return []byte(sb.String())
}

func buildMIME(msg Message) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("From: %s\r\n", msg.From))
	sb.WriteString(fmt.Sprintf("To: %s\r\n", strings.Join(msg.To, ", ")))
	if len(msg.CC) > 0 {
		sb.WriteString(fmt.Sprintf("Cc: %s\r\n", strings.Join(msg.CC, ", ")))
	}
	sb.WriteString(fmt.Sprintf("Subject: %s\r\n", msg.Subject))
	sb.WriteString("MIME-Version: 1.0\r\n")

	if len(msg.Attachments) > 0 {
		const mixedBoundary = "letrvu-mixed-001"
		sb.WriteString(fmt.Sprintf("Content-Type: multipart/mixed; boundary=%q\r\n\r\n", mixedBoundary))

		// Body part
		sb.WriteString(fmt.Sprintf("--%s\r\n", mixedBoundary))
		writeBodyPart(&sb, msg)
		sb.WriteString("\r\n")

		// Attachment parts
		for _, att := range msg.Attachments {
			ct := att.ContentType
			if ct == "" {
				ct = "application/octet-stream"
			}
			sb.WriteString(fmt.Sprintf("--%s\r\n", mixedBoundary))
			sb.WriteString(fmt.Sprintf("Content-Type: %s\r\n", ct))
			sb.WriteString(fmt.Sprintf("Content-Disposition: attachment; filename=%q\r\n", att.Filename))
			sb.WriteString("Content-Transfer-Encoding: base64\r\n\r\n")
			// RFC 2045: base64 lines must not exceed 76 characters.
			encoded := base64.StdEncoding.EncodeToString(att.Data)
			for i := 0; i < len(encoded); i += 76 {
				end := i + 76
				if end > len(encoded) {
					end = len(encoded)
				}
				sb.WriteString(encoded[i:end])
				sb.WriteString("\r\n")
			}
		}

		sb.WriteString(fmt.Sprintf("--%s--\r\n", mixedBoundary))
	} else {
		writeBodyPart(&sb, msg)
	}

	return sb.String()
}

// writeBodyPart writes the text/plain or multipart/alternative body section.
func writeBodyPart(sb *strings.Builder, msg Message) {
	if msg.HTML != "" {
		boundary := "letrvu-boundary-001"
		sb.WriteString(fmt.Sprintf("Content-Type: multipart/alternative; boundary=%q\r\n", boundary))
		sb.WriteString("\r\n")
		sb.WriteString(fmt.Sprintf("--%s\r\n", boundary))
		sb.WriteString("Content-Type: text/plain; charset=UTF-8\r\n\r\n")
		sb.WriteString(msg.Text)
		sb.WriteString("\r\n")
		sb.WriteString(fmt.Sprintf("--%s\r\n", boundary))
		sb.WriteString("Content-Type: text/html; charset=UTF-8\r\n\r\n")
		sb.WriteString(msg.HTML)
		sb.WriteString("\r\n")
		sb.WriteString(fmt.Sprintf("--%s--\r\n", boundary))
	} else {
		sb.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
		sb.WriteString("\r\n")
		sb.WriteString(msg.Text)
	}
}
