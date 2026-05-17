// Package sieve implements a minimal ManageSieve client (RFC 5804) for
// uploading vacation autoresponder scripts to a mail server.
package sieve

import (
	"bufio"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
	"time"
)

// DefaultTLSConfig is used for STARTTLS upgrades. Set InsecureSkipVerify
// in the same way as imap.DefaultTLSConfig when self-signed certs are needed.
var DefaultTLSConfig *tls.Config

// Client is a ManageSieve client (RFC 5804).
type Client struct {
	conn net.Conn
	r    *bufio.Reader
}

// Connect dials ManageSieve on port 4190 of the given host, performs STARTTLS
// if offered, and authenticates with SASL PLAIN using the given credentials.
// Returns an error (wrapping net.Error) if port 4190 is unreachable, so
// callers can detect "not supported" by checking the error.
func Connect(host, username, password string) (*Client, error) {
	conn, err := net.DialTimeout("tcp", net.JoinHostPort(host, "4190"), 5*time.Second)
	if err != nil {
		return nil, err
	}
	conn.SetDeadline(time.Now().Add(30 * time.Second)) //nolint:errcheck

	c := &Client{conn: conn, r: bufio.NewReader(conn)}

	caps, err := c.readCapabilities()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("managesieve greeting: %w", err)
	}

	// STARTTLS if offered.
	if _, ok := caps["STARTTLS"]; ok {
		if err := c.sendLine("STARTTLS"); err != nil {
			conn.Close()
			return nil, err
		}
		if err := c.readOK(); err != nil {
			conn.Close()
			return nil, fmt.Errorf("managesieve STARTTLS: %w", err)
		}
		tlsCfg := &tls.Config{ServerName: host}
		if DefaultTLSConfig != nil {
			tlsCfg.InsecureSkipVerify = DefaultTLSConfig.InsecureSkipVerify //nolint:gosec
		}
		tlsConn := tls.Client(conn, tlsCfg)
		if err := tlsConn.Handshake(); err != nil {
			tlsConn.Close()
			return nil, fmt.Errorf("managesieve TLS handshake: %w", err)
		}
		c.conn = tlsConn
		c.r = bufio.NewReader(tlsConn)
		// Re-read capabilities after upgrade.
		if _, err := c.readCapabilities(); err != nil {
			tlsConn.Close()
			return nil, fmt.Errorf("managesieve post-TLS caps: %w", err)
		}
	}

	// SASL PLAIN: \0username\0password, base64-encoded.
	ir := base64.StdEncoding.EncodeToString([]byte("\x00" + username + "\x00" + password))
	if err := c.sendLine(`AUTHENTICATE "PLAIN" "` + ir + `"`); err != nil {
		conn.Close()
		return nil, err
	}
	if err := c.readOK(); err != nil {
		conn.Close()
		return nil, fmt.Errorf("managesieve auth: %w", err)
	}

	return c, nil
}

// Close sends LOGOUT and closes the connection.
func (c *Client) Close() {
	_ = c.sendLine("LOGOUT")
	c.conn.Close()
}

// PutScript uploads a Sieve script with the given name, replacing any
// existing script with that name.
func (c *Client) PutScript(name, content string) error {
	script := []byte(content)
	// Non-synchronized literal {n+}: client sends without waiting for continuation.
	cmd := fmt.Sprintf("PUTSCRIPT %s {%d+}", quoteString(name), len(script))
	if err := c.sendLine(cmd); err != nil {
		return err
	}
	if _, err := c.conn.Write(script); err != nil {
		return err
	}
	return c.readOK()
}

// SetActive activates the named script. Pass an empty string to deactivate
// all scripts without deleting them.
func (c *Client) SetActive(name string) error {
	if err := c.sendLine("SETACTIVE " + quoteString(name)); err != nil {
		return err
	}
	return c.readOK()
}

// ScriptExists reports whether a script with the given name exists on the server.
func (c *Client) ScriptExists(name string) (bool, error) {
	if err := c.sendLine("GETSCRIPT " + quoteString(name)); err != nil {
		return false, err
	}
	line, err := c.readLine()
	if err != nil {
		return false, err
	}
	upper := strings.ToUpper(strings.TrimSpace(line))
	if strings.HasPrefix(upper, "NO") || strings.HasPrefix(upper, "BYE") {
		return false, nil
	}
	// Server sent the script as a literal; drain it then read OK.
	if strings.HasPrefix(line, "{") {
		if err := c.drainLiteral(line); err != nil {
			return false, err
		}
	}
	return true, c.readOK()
}

// ── internal helpers ──────────────────────────────────────────────────────────

func (c *Client) sendLine(s string) error {
	_, err := fmt.Fprintf(c.conn, "%s\r\n", s)
	return err
}

func (c *Client) readLine() (string, error) {
	line, err := c.r.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimRight(line, "\r\n"), nil
}

// readOK reads response lines until OK. Returns an error for NO or BYE.
func (c *Client) readOK() error {
	for {
		line, err := c.readLine()
		if err != nil {
			return err
		}
		upper := strings.ToUpper(strings.TrimSpace(line))
		switch {
		case strings.HasPrefix(upper, "OK"):
			return nil
		case strings.HasPrefix(upper, "NO"), strings.HasPrefix(upper, "BYE"):
			return fmt.Errorf("managesieve: %s", line)
		}
		// Intermediate line — ignore.
	}
}

// readCapabilities reads the server greeting capability block until OK.
func (c *Client) readCapabilities() (map[string]string, error) {
	caps := make(map[string]string)
	for {
		line, err := c.readLine()
		if err != nil {
			return nil, err
		}
		upper := strings.ToUpper(strings.TrimSpace(line))
		if strings.HasPrefix(upper, "OK") {
			return caps, nil
		}
		if strings.HasPrefix(upper, "NO") || strings.HasPrefix(upper, "BYE") {
			return nil, fmt.Errorf("managesieve: %s", line)
		}
		name, rest := parseQuotedString(line)
		val := ""
		if rest != "" {
			val, _ = parseQuotedString(strings.TrimSpace(rest))
		}
		caps[strings.ToUpper(name)] = val
	}
}

// drainLiteral reads and discards a server literal value given its descriptor
// line (e.g. "{1234}"). RFC 5804: after the n bytes, there is a CRLF.
func (c *Client) drainLiteral(line string) error {
	end := strings.IndexByte(line, '}')
	if end < 0 {
		return fmt.Errorf("managesieve: malformed literal: %s", line)
	}
	sizeStr := strings.TrimRight(line[1:end], "+")
	size, err := strconv.Atoi(sizeStr)
	if err != nil {
		return fmt.Errorf("managesieve: bad literal size: %w", err)
	}
	if _, err := io.CopyN(io.Discard, c.r, int64(size)); err != nil {
		return err
	}
	// Consume trailing CRLF (RFC 5804 sieve-script CRLF).
	_, _ = c.r.ReadString('\n')
	return nil
}

// quoteString returns a ManageSieve quoted string.
func quoteString(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	return `"` + s + `"`
}

// parseQuotedString parses a leading quoted string from s and returns the
// content and the remainder of the line after the closing quote. If s does
// not start with '"', an unquoted atom (up to the first whitespace) is returned.
func parseQuotedString(s string) (string, string) {
	if len(s) == 0 {
		return "", ""
	}
	if s[0] != '"' {
		// Unquoted atom.
		i := strings.IndexAny(s, " \t")
		if i < 0 {
			return s, ""
		}
		return s[:i], s[i:]
	}
	var result strings.Builder
	i := 1
	for i < len(s) {
		if s[i] == '\\' && i+1 < len(s) {
			result.WriteByte(s[i+1])
			i += 2
			continue
		}
		if s[i] == '"' {
			return result.String(), s[i+1:]
		}
		result.WriteByte(s[i])
		i++
	}
	return result.String(), ""
}
