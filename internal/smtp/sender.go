// Package smtp handles outbound mail delivery.
package smtp

import (
	"fmt"
	"net/smtp"
	"strings"
)

// Config holds the SMTP server connection details.
type Config struct {
	Host     string
	Port     int
	Username string
	Password string
}

// Message is an outbound email.
type Message struct {
	From    string
	To      []string
	CC      []string
	Subject string
	Text    string // plain text body
	HTML    string // optional HTML body; if set, sends multipart/alternative
}

// Send delivers msg via SMTP STARTTLS with PLAIN auth.
func Send(cfg Config, msg Message) error {
	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	auth := smtp.PlainAuth("", cfg.Username, cfg.Password, cfg.Host)

	recipients := append(msg.To, msg.CC...)
	body := buildMIME(msg)

	if err := smtp.SendMail(addr, auth, msg.From, recipients, []byte(body)); err != nil {
		return fmt.Errorf("smtp send: %w", err)
	}
	return nil
}

func buildMIME(msg Message) string {
	var sb strings.Builder

	allTo := strings.Join(msg.To, ", ")
	sb.WriteString(fmt.Sprintf("From: %s\r\n", msg.From))
	sb.WriteString(fmt.Sprintf("To: %s\r\n", allTo))
	if len(msg.CC) > 0 {
		sb.WriteString(fmt.Sprintf("Cc: %s\r\n", strings.Join(msg.CC, ", ")))
	}
	sb.WriteString(fmt.Sprintf("Subject: %s\r\n", msg.Subject))
	sb.WriteString("MIME-Version: 1.0\r\n")

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

	return sb.String()
}
