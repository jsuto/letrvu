// Package smtp handles outbound mail delivery.
package smtp

import (
	"crypto/rand"
	"crypto/tls"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"log"
	netsmtp "net/smtp"
	"regexp"
	"strings"
	"time"
)

// Debug enables verbose logging for development. Set via LOG_LEVEL=debug.
var Debug bool

func debugf(format string, args ...any) {
	if Debug {
		log.Printf("[smtp debug] "+format, args...)
	}
}

// DefaultTLSConfig is used for every outbound SMTP connection. main() may
// replace it before the server starts (e.g. to toggle InsecureSkipVerify).
var DefaultTLSConfig = &tls.Config{
	InsecureSkipVerify: true, //nolint:gosec
}

// Hostname is used as the right-hand side of generated Message-IDs.
// Set it to the public hostname of the webmail server (e.g. "mail.example.com").
// Defaults to "localhost" if not configured.
var Hostname = "localhost"

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

	// Threading / identification headers (RFC 2822).
	// MessageID is used as-is if non-empty; otherwise one is auto-generated
	// from Hostname. Callers should pre-generate it so the same ID can be
	// stored in the Sent folder via IMAP APPEND.
	MessageID  string
	InReplyTo  string // Message-ID of the message being replied to
	References string // space-separated chain of Message-IDs for thread tracking

	// PGP/MIME signing (RFC 3156). When PGPSignature is set and there are no
	// attachments the message is wrapped in multipart/signed. With attachments
	// the signature is included as a signature.asc attachment instead.
	PGPSignature string // armored detached PGP signature
	PGPMicAlg    string // hash algorithm string, e.g. "pgp-sha512"
}

// Send delivers msg via SMTP. Port 465 uses implicit TLS (SMTPS — the TLS
// handshake happens immediately on connect). All other ports use STARTTLS
// (plain connection that upgrades to TLS after the initial greeting).
func Send(cfg Config, msg Message) error {
	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)

	tlsCfg := DefaultTLSConfig.Clone()
	tlsCfg.ServerName = cfg.Host

	var c *netsmtp.Client
	var err error

	if cfg.Port == 465 {
		// Implicit TLS (SMTPS): TLS wraps the connection from the first byte.
		conn, dialErr := tls.Dial("tcp", addr, tlsCfg)
		if dialErr != nil {
			return fmt.Errorf("smtp dial tls: %w", dialErr)
		}
		c, err = netsmtp.NewClient(conn, cfg.Host)
		if err != nil {
			conn.Close()
			return fmt.Errorf("smtp client: %w", err)
		}
	} else {
		// STARTTLS: plain TCP first, then upgrade.
		c, err = netsmtp.Dial(addr)
		if err != nil {
			return fmt.Errorf("smtp dial: %w", err)
		}
		if err = c.StartTLS(tlsCfg); err != nil {
			c.Close()
			return fmt.Errorf("smtp starttls: %w", err)
		}
	}
	defer c.Close()

	// Identify ourselves with a proper hostname instead of the default "localhost".
	if err = c.Hello(Hostname); err != nil {
		return fmt.Errorf("smtp ehlo: %w", err)
	}

	auth := netsmtp.PlainAuth("", cfg.Username, cfg.Password, cfg.Host)
	if err = c.Auth(auth); err != nil {
		return fmt.Errorf("smtp auth: %w", err)
	}

	envelopeFrom := msg.EnvelopeFrom
	if envelopeFrom == "" {
		envelopeFrom = msg.From
	}

	debugf("sending from=%s to=[%s]", envelopeFrom, strings.Join(append(msg.To, msg.CC...), ", "))

	if err = c.Mail(envelopeFrom); err != nil {
		return fmt.Errorf("smtp mail from: %w", err)
	}

	for _, rcpt := range append(msg.To, msg.CC...) {
		if err = c.Rcpt(rcpt); err != nil {
			return fmt.Errorf("smtp rcpt %s: %w", rcpt, err)
		}
	}

	wc, err := c.Data()
	if err != nil {
		return fmt.Errorf("smtp data: %w", err)
	}
	if _, err = fmt.Fprint(wc, buildMIME(msg)); err != nil {
		wc.Close()
		return fmt.Errorf("smtp write: %w", err)
	}
	if err = wc.Close(); err != nil {
		return fmt.Errorf("smtp data close: %w", err)
	}

	return c.Quit()
}

// BuildRFC822 returns the complete RFC 2822 message bytes including all
// headers (Date, Message-ID, X-Mailer, etc.). Used to save drafts and sent
// copies via IMAP APPEND. Callers should pre-populate msg.MessageID so that
// the stored copy matches what was handed to Send.
func BuildRFC822(msg Message) []byte {
	return []byte(buildMIME(msg))
}

// randomToken returns n random bytes encoded as lowercase hex.
func randomToken(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(b)
}

func buildMIME(msg Message) string {
	var sb strings.Builder

	// RFC 2822 §3.6: Date and From are required.
	sb.WriteString(fmt.Sprintf("Date: %s\r\n", time.Now().Format(time.RFC1123Z)))

	// Message-ID: use the pre-generated one if present so Send and the Sent
	// IMAP copy carry the same ID; otherwise auto-generate.
	msgID := msg.MessageID
	if msgID == "" {
		msgID = fmt.Sprintf("<%d.%s@%s>", time.Now().UnixNano(), randomToken(8), Hostname)
	}
	sb.WriteString(fmt.Sprintf("Message-ID: %s\r\n", msgID))

	if msg.InReplyTo != "" {
		sb.WriteString(fmt.Sprintf("In-Reply-To: %s\r\n", msg.InReplyTo))
	}
	if msg.References != "" {
		sb.WriteString(fmt.Sprintf("References: %s\r\n", msg.References))
	}

	sb.WriteString(fmt.Sprintf("From: %s\r\n", msg.From))
	sb.WriteString(fmt.Sprintf("To: %s\r\n", strings.Join(msg.To, ", ")))
	if len(msg.CC) > 0 {
		sb.WriteString(fmt.Sprintf("Cc: %s\r\n", strings.Join(msg.CC, ", ")))
	}
	sb.WriteString(fmt.Sprintf("Subject: %s\r\n", msg.Subject))
	sb.WriteString("X-Mailer: letrvu\r\n")
	sb.WriteString("MIME-Version: 1.0\r\n")

	if msg.PGPSignature != "" && len(msg.Attachments) == 0 {
		writePGPMIMESigned(&sb, msg)
		return sb.String()
	}

	if msg.PGPSignature != "" {
		// Has attachments: append signature.asc so it travels with them.
		msg.Attachments = append(msg.Attachments, Attachment{
			Filename:    "signature.asc",
			ContentType: "application/pgp-signature",
			Data:        []byte(msg.PGPSignature),
		})
	}

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

			// RFC 2045 §6.4: message/* types MUST NOT use base64 or
			// quoted-printable. Embed the raw bytes inline with 8bit encoding
			// so the inner message is handed to recipients verbatim — including
			// all original headers (e.g. spam-filter watermarks).
			if strings.HasPrefix(strings.ToLower(ct), "message/") {
				sb.WriteString("Content-Transfer-Encoding: 8bit\r\n\r\n")
				sb.Write(att.Data) //nolint:errcheck
				sb.WriteString("\r\n")
				continue
			}

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
		text := msg.Text
		if text == "" {
			text = stripHTML(msg.HTML)
		}
		boundary := "letrvu-boundary-001"
		sb.WriteString(fmt.Sprintf("Content-Type: multipart/alternative; boundary=%q\r\n", boundary))
		sb.WriteString("\r\n")
		sb.WriteString(fmt.Sprintf("--%s\r\n", boundary))
		sb.WriteString("Content-Type: text/plain; charset=UTF-8\r\n\r\n")
		sb.WriteString(text)
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

// stripHTML converts HTML to a plain-text approximation for the text/plain
// part of a multipart/alternative message.
var (
	reBlockEnd = strings.NewReplacer(
		"</p>", "\n", "</div>", "\n", "</li>", "\n",
		"</h1>", "\n", "</h2>", "\n", "</h3>", "\n",
		"</h4>", "\n", "</h5>", "\n", "</h6>", "\n",
		"</blockquote>", "\n",
	)
	reTags        = regexp.MustCompile(`<[^>]+>`)
	reExcessLines = regexp.MustCompile(`\n{3,}`)
	htmlEntities  = strings.NewReplacer(
		"&amp;", "&", "&lt;", "<", "&gt;", ">",
		"&quot;", `"`, "&#39;", "'", "&nbsp;", " ",
	)
)

// normalizeCRLF converts all line endings to CRLF, as required for canonical
// MIME content (RFC 2822 §2.3, RFC 3156 §5).
func normalizeCRLF(s string) string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")
	return strings.ReplaceAll(s, "\n", "\r\n")
}

// writePGPMIMESigned outputs a multipart/signed body (RFC 3156). The first
// part is the canonical text/plain body signed by the sender; the second is
// the armored detached PGP signature. The canonical body part MUST be
// byte-identical to what the sender signed — see signMIME() in pgp.js.
func writePGPMIMESigned(sb *strings.Builder, msg Message) {
	micalg := msg.PGPMicAlg
	if micalg == "" {
		micalg = "pgp-sha512"
	}
	const boundary = "letrvu-pgpsig-001"
	sb.WriteString(fmt.Sprintf(
		"Content-Type: multipart/signed; protocol=\"application/pgp-signature\";\r\n\tmicalg=%s; boundary=%q\r\n\r\n",
		micalg, boundary,
	))

	// Part 1 — signed body. Headers and body must match exactly what the
	// frontend passed to openpgp.sign() in signMIME().
	sb.WriteString(fmt.Sprintf("--%s\r\n", boundary))
	sb.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
	sb.WriteString("Content-Transfer-Encoding: 8bit\r\n\r\n")
	sb.WriteString(normalizeCRLF(msg.Text))
	sb.WriteString("\r\n")

	// Part 2 — detached signature
	sb.WriteString(fmt.Sprintf("--%s\r\n", boundary))
	sb.WriteString("Content-Type: application/pgp-signature\r\n")
	sb.WriteString("Content-Description: OpenPGP digital signature\r\n")
	sb.WriteString("Content-Disposition: attachment; filename=\"signature.asc\"\r\n\r\n")
	sb.WriteString(msg.PGPSignature)
	sb.WriteString("\r\n")
	sb.WriteString(fmt.Sprintf("--%s--\r\n", boundary))
}

func stripHTML(html string) string {
	s := reBlockEnd.Replace(html)
	// br tags need a regex due to optional attributes and self-closing slash
	s = regexp.MustCompile(`(?i)<br\s*/?>|<hr\s*/?>`).ReplaceAllString(s, "\n")
	s = reTags.ReplaceAllString(s, "")
	s = htmlEntities.Replace(s)
	s = reExcessLines.ReplaceAllString(s, "\n\n")
	return strings.TrimSpace(s)
}
