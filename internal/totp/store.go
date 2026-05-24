package totp

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	"time"

	"github.com/jsuto/letrvu/internal/db"
	"golang.org/x/crypto/hkdf"
)

// Store persists TOTP credentials and recovery codes.
type Store struct {
	db     *db.DB
	secret []byte
}

// NewStore creates a Store using the given database and server secret.
// The secret is used to encrypt TOTP secrets at rest.
func NewStore(database *db.DB, secret []byte) *Store {
	return &Store{db: database, secret: secret}
}

// IsEnabled reports whether TOTP is active for the given user.
func (s *Store) IsEnabled(username, imapHost string) bool {
	var enabledAt string
	err := s.db.QueryRow(
		s.db.Q(`SELECT enabled_at FROM totp_credentials WHERE username = ? AND imap_host = ?`),
		username, imapHost,
	).Scan(&enabledAt)
	return err == nil && enabledAt != ""
}

// GetSecret returns the decrypted TOTP secret for the user, and whether one exists.
// A row with an empty enabled_at is a pending enrollment (not yet active).
func (s *Store) GetSecret(username, imapHost string) (secret string, enabled bool, ok bool) {
	var encSecret, enabledAt string
	err := s.db.QueryRow(
		s.db.Q(`SELECT encrypted_secret, enabled_at FROM totp_credentials WHERE username = ? AND imap_host = ?`),
		username, imapHost,
	).Scan(&encSecret, &enabledAt)
	if err != nil {
		return "", false, false
	}
	dec, err := s.decryptSecret(encSecret)
	if err != nil {
		return "", false, false
	}
	return dec, enabledAt != "", true
}

// SavePendingSecret stores an unconfirmed TOTP secret. The user must verify a
// code before calling Enable to activate it.
func (s *Store) SavePendingSecret(username, imapHost, secret string) error {
	enc, err := s.encryptSecret(secret)
	if err != nil {
		return fmt.Errorf("encrypt totp secret: %w", err)
	}
	_, err = s.db.Exec(
		s.db.Q(`INSERT INTO totp_credentials (username, imap_host, encrypted_secret, enabled_at)
			VALUES (?, ?, ?, '')
			ON CONFLICT (username, imap_host) DO UPDATE SET encrypted_secret = excluded.encrypted_secret, enabled_at = ''`),
		username, imapHost, enc,
	)
	return err
}

// Enable marks the TOTP credential as active. Call after the user has
// successfully verified the enrollment code.
func (s *Store) Enable(username, imapHost string) error {
	_, err := s.db.Exec(
		s.db.Q(`UPDATE totp_credentials SET enabled_at = ? WHERE username = ? AND imap_host = ?`),
		time.Now().UTC().Format(time.RFC3339), username, imapHost,
	)
	return err
}

// Delete removes all TOTP credentials and recovery codes for the user.
func (s *Store) Delete(username, imapHost string) error {
	_, err := s.db.Exec(
		s.db.Q(`DELETE FROM totp_credentials WHERE username = ? AND imap_host = ?`),
		username, imapHost,
	)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(
		s.db.Q(`DELETE FROM totp_recovery_codes WHERE username = ? AND imap_host = ?`),
		username, imapHost,
	)
	return err
}

// SaveRecoveryCodes replaces all recovery codes for the user with new hashes.
func (s *Store) SaveRecoveryCodes(username, imapHost string, hashes []string) error {
	_, err := s.db.Exec(
		s.db.Q(`DELETE FROM totp_recovery_codes WHERE username = ? AND imap_host = ?`),
		username, imapHost,
	)
	if err != nil {
		return err
	}
	for _, h := range hashes {
		if _, err := s.db.Exec(
			s.db.Q(`INSERT INTO totp_recovery_codes (username, imap_host, code_hash, used_at) VALUES (?, ?, ?, '')`),
			username, imapHost, h,
		); err != nil {
			return err
		}
	}
	return nil
}

// ConsumeRecoveryCode checks whether the submitted code matches an unused
// recovery code for the user, and if so marks it as used.
// Returns true if a valid code was consumed.
func (s *Store) ConsumeRecoveryCode(username, imapHost, code string) bool {
	h := HashRecoveryCode(code)
	res, err := s.db.Exec(
		s.db.Q(`UPDATE totp_recovery_codes
			SET used_at = ?
			WHERE username = ? AND imap_host = ? AND code_hash = ? AND used_at = ''`),
		time.Now().UTC().Format(time.RFC3339), username, imapHost, h,
	)
	if err != nil {
		return false
	}
	n, _ := res.RowsAffected()
	return n > 0
}

// RemainingRecoveryCodes returns the count of unused recovery codes.
func (s *Store) RemainingRecoveryCodes(username, imapHost string) int {
	var n int
	s.db.QueryRow( //nolint:errcheck
		s.db.Q(`SELECT COUNT(*) FROM totp_recovery_codes WHERE username = ? AND imap_host = ? AND used_at = ''`),
		username, imapHost,
	).Scan(&n)
	return n
}

// encryptSecret encrypts a TOTP secret with AES-256-GCM.
// A fresh random nonce is derived into the key via HKDF; the nonce is
// prepended to the ciphertext and the whole thing is base64-encoded.
func (s *Store) encryptSecret(plaintext string) (string, error) {
	nonce := make([]byte, 16)
	if _, err := rand.Read(nonce); err != nil {
		return "", err
	}
	key, err := s.deriveKey(nonce)
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	gcmNonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(gcmNonce); err != nil {
		return "", err
	}
	ciphertext := gcm.Seal(gcmNonce, gcmNonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(append(nonce, ciphertext...)), nil
}

func (s *Store) decryptSecret(encoded string) (string, error) {
	raw, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", fmt.Errorf("decode: %w", err)
	}
	if len(raw) < 16 {
		return "", fmt.Errorf("ciphertext too short")
	}
	nonce, ciphertext := raw[:16], raw[16:]
	key, err := s.deriveKey(nonce)
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return "", fmt.Errorf("ciphertext too short")
	}
	plaintext, err := gcm.Open(nil, ciphertext[:nonceSize], ciphertext[nonceSize:], nil)
	if err != nil {
		return "", fmt.Errorf("decrypt: %w", err)
	}
	return string(plaintext), nil
}

func (s *Store) deriveKey(nonce []byte) ([]byte, error) {
	r := hkdf.New(sha256.New, s.secret, nonce, []byte("letrvu-totp-key"))
	key := make([]byte, 32)
	if _, err := io.ReadFull(r, key); err != nil {
		return nil, err
	}
	return key, nil
}

// encryptForPending is an exported-to-package helper used by session.Store
// to encrypt the IMAP password for a pending login row. It uses the same
// AES-256-GCM + HKDF scheme as session passwords.
func encryptForPending(secret, nonce []byte, plaintext string) (string, error) {
	r := hkdf.New(sha256.New, secret, nonce, []byte("letrvu-pending-key"))
	key := make([]byte, 32)
	if _, err := io.ReadFull(r, key); err != nil {
		return "", err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	gcmNonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(gcmNonce); err != nil {
		return "", err
	}
	ct := gcm.Seal(gcmNonce, gcmNonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ct), nil
}

func decryptForPending(secret, nonce []byte, encoded string) (string, error) {
	r := hkdf.New(sha256.New, secret, nonce, []byte("letrvu-pending-key"))
	key := make([]byte, 32)
	if _, err := io.ReadFull(r, key); err != nil {
		return "", err
	}
	ct, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonceSize := gcm.NonceSize()
	if len(ct) < nonceSize {
		return "", fmt.Errorf("ciphertext too short")
	}
	pt, err := gcm.Open(nil, ct[:nonceSize], ct[nonceSize:], nil)
	if err != nil {
		return "", fmt.Errorf("decrypt: %w", err)
	}
	return string(pt), nil
}

// PendingLogin holds partially-completed login credentials awaiting TOTP verification.
type PendingLogin struct {
	Username  string
	IMAPHost  string
	IMAPPort  int
	SMTPHost  string
	SMTPPort  int
	Password  string
	UserAgent string
}

// PendingStore manages pending logins in the totp_pending table.
// It is separate from session.Store to keep the TOTP package self-contained.
type PendingStore struct {
	db     *db.DB
	secret []byte
}

// NewPendingStore creates a PendingStore.
func NewPendingStore(database *db.DB, secret []byte) *PendingStore {
	return &PendingStore{db: database, secret: secret}
}

// Create stores a pending login and returns a cookie value in the form "id.nonceHex".
func (ps *PendingStore) Create(imapHost string, imapPort int, smtpHost string, smtpPort int, username, password, userAgent string) (string, error) {
	id, err := randomHex(16)
	if err != nil {
		return "", err
	}
	nonceBytes, err := randomBytes(16)
	if err != nil {
		return "", err
	}

	encPwd, err := encryptForPending(ps.secret, nonceBytes, password)
	if err != nil {
		return "", fmt.Errorf("encrypt pending password: %w", err)
	}

	expiresAt := time.Now().UTC().Add(10 * time.Minute).Format(time.RFC3339)
	_, err = ps.db.Exec(
		ps.db.Q(`INSERT INTO totp_pending
			(id, username, imap_host, imap_port, smtp_host, smtp_port, encrypted_password, user_agent, expires_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`),
		id, username, imapHost, imapPort, smtpHost, smtpPort, encPwd, userAgent, expiresAt,
	)
	if err != nil {
		return "", fmt.Errorf("insert pending: %w", err)
	}

	return id + "." + hex.EncodeToString(nonceBytes), nil
}

// Get retrieves and decrypts a pending login. Returns nil, false if not found or expired.
func (ps *PendingStore) Get(cookieValue string) (*PendingLogin, bool) {
	id, nonceHex, ok := parseCookieValue(cookieValue)
	if !ok {
		return nil, false
	}
	nonce, err := hex.DecodeString(nonceHex)
	if err != nil {
		return nil, false
	}

	var p PendingLogin
	var encPwd, expiresAtStr string
	err = ps.db.QueryRow(
		ps.db.Q(`SELECT username, imap_host, imap_port, smtp_host, smtp_port, encrypted_password, user_agent, expires_at
			FROM totp_pending WHERE id = ?`),
		id,
	).Scan(&p.Username, &p.IMAPHost, &p.IMAPPort, &p.SMTPHost, &p.SMTPPort, &encPwd, &p.UserAgent, &expiresAtStr)
	if err != nil {
		return nil, false
	}

	expiresAt, err := time.Parse(time.RFC3339, expiresAtStr)
	if err != nil || time.Now().UTC().After(expiresAt) {
		ps.Delete(cookieValue)
		return nil, false
	}

	p.Password, err = decryptForPending(ps.secret, nonce, encPwd)
	if err != nil {
		return nil, false
	}
	return &p, true
}

// Delete removes a pending login row.
func (ps *PendingStore) Delete(cookieValue string) {
	id, _, ok := parseCookieValue(cookieValue)
	if !ok {
		return
	}
	ps.db.Exec(ps.db.Q(`DELETE FROM totp_pending WHERE id = ?`), id) //nolint:errcheck
}

// DeleteExpired removes all expired pending logins.
func (ps *PendingStore) DeleteExpired() {
	now := time.Now().UTC().Format(time.RFC3339)
	ps.db.Exec(ps.db.Q(`DELETE FROM totp_pending WHERE expires_at < ?`), now) //nolint:errcheck
}

func parseCookieValue(v string) (id, nonce string, ok bool) {
	// Format: "hexID.hexNonce"
	dot := -1
	for i := len(v) - 1; i >= 0; i-- {
		if v[i] == '.' {
			dot = i
			break
		}
	}
	if dot <= 0 || dot == len(v)-1 {
		return "", "", false
	}
	return v[:dot], v[dot+1:], true
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
