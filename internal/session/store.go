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
	ID             string
	Username       string
	IMAPHost       string
	IMAPPort       int
	SMTPHost       string
	SMTPPort       int
	Password       string
	UserAgent      string
	CreatedAt      time.Time
	LastActivityAt time.Time
}

// SessionInfo is a sanitised view of a session for the active-sessions list.
type SessionInfo struct {
	ID             string    `json:"id"`
	UserAgent      string    `json:"user_agent"`
	CreatedAt      time.Time `json:"created_at"`
	LastActivityAt time.Time `json:"last_activity_at"`
	Current        bool      `json:"current"`
}

// Store is a database-backed session registry.
type Store struct {
	db          *db.DB
	secret      []byte
	idleTimeout time.Duration // 0 means disabled
}

// NewStore creates a Store using the given database and server secret.
// secret should be 32 random bytes. If SESSION_SECRET is not configured,
// main() generates an ephemeral secret and warns the operator.
// idleTimeout is the maximum inactivity period before a session expires;
// 0 disables idle-based expiry (sessions only expire at their absolute expiry).
func NewStore(database *db.DB, secret []byte, idleTimeout time.Duration) *Store {
	return &Store{db: database, secret: secret, idleTimeout: idleTimeout}
}

// Create encrypts the IMAP password, persists the session, and returns the
// cookie value in the form "<sessionID>.<clientNonceHex>".
func (s *Store) Create(imapHost string, imapPort int, smtpHost string, smtpPort int, username, password, userAgent string) (string, error) {
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
			(id, username, imap_host, imap_port, smtp_host, smtp_port, encrypted_password, created_at, expires_at, user_agent, last_activity_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`),
		id, username, imapHost, imapPort, smtpHost, smtpPort, encPwd,
		now.Format(time.RFC3339),
		now.Add(sessionExpiry).Format(time.RFC3339),
		userAgent,
		now.Format(time.RFC3339),
	)
	if err != nil {
		return "", fmt.Errorf("insert session: %w", err)
	}

	return id + "." + hex.EncodeToString(nonceBytes), nil
}

// Get retrieves and decrypts a session using the cookie value.
// Returns nil, false if the cookie is malformed, the session is not found,
// or it has expired (absolute or idle timeout).
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
	var encPwd, createdAtStr, expiresAtStr, lastActivityAtStr string
	err = s.db.QueryRow(
		s.db.Q(`SELECT id, username, imap_host, imap_port, smtp_host, smtp_port,
			encrypted_password, created_at, expires_at, user_agent, last_activity_at
			FROM sessions WHERE id = ?`),
		id,
	).Scan(
		&sess.ID, &sess.Username, &sess.IMAPHost, &sess.IMAPPort,
		&sess.SMTPHost, &sess.SMTPPort, &encPwd, &createdAtStr, &expiresAtStr,
		&sess.UserAgent, &lastActivityAtStr,
	)
	if err != nil {
		return nil, false
	}

	now := time.Now().UTC()

	expiresAt, err := time.Parse(time.RFC3339, expiresAtStr)
	if err != nil || now.After(expiresAt) {
		s.Delete(cookieValue)
		return nil, false
	}

	// Idle timeout check: if last_activity_at is set and too old, expire the session.
	if s.idleTimeout > 0 && lastActivityAtStr != "" {
		lastActivity, err := time.Parse(time.RFC3339, lastActivityAtStr)
		if err == nil && now.Sub(lastActivity) > s.idleTimeout {
			s.Delete(cookieValue)
			return nil, false
		}
	}

	sess.Password, err = decryptPassword(s.secret, nonceBytes, encPwd)
	if err != nil {
		return nil, false
	}
	sess.CreatedAt, _ = time.Parse(time.RFC3339, createdAtStr)
	sess.LastActivityAt, _ = time.Parse(time.RFC3339, lastActivityAtStr)

	// Lazily update last_activity_at at most once per minute to avoid a DB
	// write on every single API request while keeping timestamps accurate.
	if now.Sub(sess.LastActivityAt) > time.Minute {
		nowStr := now.Format(time.RFC3339)
		s.db.Exec(s.db.Q(`UPDATE sessions SET last_activity_at = ? WHERE id = ?`), nowStr, id) //nolint:errcheck
		sess.LastActivityAt = now
	}

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
	now := time.Now().UTC().Format(time.RFC3339)
	s.db.Exec( //nolint:errcheck
		s.db.Q(`DELETE FROM sessions WHERE expires_at < ?`),
		now,
	)
	if s.idleTimeout > 0 {
		cutoff := time.Now().UTC().Add(-s.idleTimeout).Format(time.RFC3339)
		s.db.Exec( //nolint:errcheck
			s.db.Q(`DELETE FROM sessions WHERE last_activity_at != '' AND last_activity_at < ?`),
			cutoff,
		)
	}
}

// List returns all active sessions for the given user, marking the current one.
func (s *Store) List(username, imapHost, currentID string) ([]SessionInfo, error) {
	rows, err := s.db.Query(
		s.db.Q(`SELECT id, user_agent, created_at, last_activity_at
			FROM sessions WHERE username = ? AND imap_host = ? AND expires_at > ?
			ORDER BY last_activity_at DESC`),
		username, imapHost, time.Now().UTC().Format(time.RFC3339),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []SessionInfo
	for rows.Next() {
		var si SessionInfo
		var createdAtStr, lastActivityAtStr string
		if err := rows.Scan(&si.ID, &si.UserAgent, &createdAtStr, &lastActivityAtStr); err != nil {
			continue
		}
		si.CreatedAt, _ = time.Parse(time.RFC3339, createdAtStr)
		si.LastActivityAt, _ = time.Parse(time.RFC3339, lastActivityAtStr)
		si.Current = si.ID == currentID
		sessions = append(sessions, si)
	}
	return sessions, rows.Err()
}

// DeleteAllForUser removes all sessions for the given user except the one
// identified by exceptID. Used for "logout all other devices".
// If exceptID is empty, all sessions are removed.
func (s *Store) DeleteAllForUser(username, imapHost, exceptID string) {
	if exceptID == "" {
		s.db.Exec( //nolint:errcheck
			s.db.Q(`DELETE FROM sessions WHERE username = ? AND imap_host = ?`),
			username, imapHost,
		)
		return
	}
	s.db.Exec( //nolint:errcheck
		s.db.Q(`DELETE FROM sessions WHERE username = ? AND imap_host = ? AND id != ?`),
		username, imapHost, exceptID,
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
