// Package session manages authenticated user sessions backed by a SQL database.
// IMAP passwords are stored encrypted; the encryption key is derived from a
// server-side secret combined with a per-client nonce stored in the browser
// cookie. Neither the database alone nor the cookie alone is sufficient to
// recover the password.
package session

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/jsuto/letrvu/internal/db"
)

const sessionExpiry = 7 * 24 * time.Hour

// Session holds decrypted credentials for one logged-in user.
// The password is never written to disk in plaintext.
type Session struct {
	ID        string
	Username  string
	IMAPHost  string
	IMAPPort  int
	SMTPHost  string
	SMTPPort  int
	Password  string
	CreatedAt time.Time
}

// Store is a database-backed session registry.
type Store struct {
	db     *db.DB
	secret []byte
}

// NewStore creates a Store using the given database and server secret.
// secret should be 32 random bytes. If SESSION_SECRET is not configured,
// main() generates an ephemeral secret and warns the operator.
func NewStore(database *db.DB, secret []byte) *Store {
	return &Store{db: database, secret: secret}
}

// Create encrypts the IMAP password, persists the session, and returns the
// cookie value in the form "<sessionID>.<clientNonceHex>".
func (s *Store) Create(imapHost string, imapPort int, smtpHost string, smtpPort int, username, password string) (string, error) {
	id, err := randomHex(16)
	if err != nil {
		return "", err
	}
	nonceBytes, err := randomBytes(16)
	if err != nil {
		return "", err
	}

	encPwd, err := encryptPassword(s.secret, nonceBytes, password)
	if err != nil {
		return "", fmt.Errorf("encrypt: %w", err)
	}

	now := time.Now().UTC()
	_, err = s.db.Exec(
		s.db.Q(`INSERT INTO sessions
			(id, username, imap_host, imap_port, smtp_host, smtp_port, encrypted_password, created_at, expires_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`),
		id, username, imapHost, imapPort, smtpHost, smtpPort, encPwd,
		now.Format(time.RFC3339),
		now.Add(sessionExpiry).Format(time.RFC3339),
	)
	if err != nil {
		return "", fmt.Errorf("insert session: %w", err)
	}

	return id + "." + hex.EncodeToString(nonceBytes), nil
}

// Get retrieves and decrypts a session using the cookie value.
// Returns nil, false if the cookie is malformed, the session is not found,
// or it has expired.
func (s *Store) Get(cookieValue string) (*Session, bool) {
	id, nonceHex, ok := parseCookieValue(cookieValue)
	if !ok {
		return nil, false
	}
	nonceBytes, err := hex.DecodeString(nonceHex)
	if err != nil {
		return nil, false
	}

	var sess Session
	var encPwd, createdAtStr, expiresAtStr string
	err = s.db.QueryRow(
		s.db.Q(`SELECT id, username, imap_host, imap_port, smtp_host, smtp_port,
			encrypted_password, created_at, expires_at
			FROM sessions WHERE id = ?`),
		id,
	).Scan(
		&sess.ID, &sess.Username, &sess.IMAPHost, &sess.IMAPPort,
		&sess.SMTPHost, &sess.SMTPPort, &encPwd, &createdAtStr, &expiresAtStr,
	)
	if err != nil {
		return nil, false
	}

	expiresAt, err := time.Parse(time.RFC3339, expiresAtStr)
	if err != nil || time.Now().UTC().After(expiresAt) {
		s.Delete(cookieValue)
		return nil, false
	}

	sess.Password, err = decryptPassword(s.secret, nonceBytes, encPwd)
	if err != nil {
		return nil, false
	}
	sess.CreatedAt, _ = time.Parse(time.RFC3339, createdAtStr)
	return &sess, true
}

// Delete removes a session from the database, invalidating the cookie.
func (s *Store) Delete(cookieValue string) {
	id, _, ok := parseCookieValue(cookieValue)
	if !ok {
		return
	}
	s.db.Exec(s.db.Q(`DELETE FROM sessions WHERE id = ?`), id) //nolint:errcheck
}

// DeleteExpired removes all expired sessions. Call on startup and periodically.
func (s *Store) DeleteExpired() {
	s.db.Exec( //nolint:errcheck
		s.db.Q(`DELETE FROM sessions WHERE expires_at < ?`),
		time.Now().UTC().Format(time.RFC3339),
	)
}

func parseCookieValue(v string) (id, nonce string, ok bool) {
	parts := strings.SplitN(v, ".", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", false
	}
	return parts[0], parts[1], true
}

func randomHex(n int) (string, error) {
	b, err := randomBytes(n)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func randomBytes(n int) ([]byte, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return nil, err
	}
	return b, nil
}
