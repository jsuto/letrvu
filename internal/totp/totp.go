// Package totp implements TOTP (RFC 6238) helpers for two-factor authentication.
package totp

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"image/png"
	"strings"
	"time"

	gotp "github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"
)

// Generate creates a new TOTP key for the given user.
// Returns the base32 secret, the otpauth:// URL, and a 200×200 QR PNG.
func Generate(issuer, accountName string) (secret, otpauthURL string, qrPNG []byte, err error) {
	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      issuer,
		AccountName: accountName,
		Algorithm:   gotp.AlgorithmSHA1,
		Digits:      gotp.DigitsSix,
		Period:      30,
	})
	if err != nil {
		return "", "", nil, fmt.Errorf("generate totp key: %w", err)
	}

	img, err := key.Image(200, 200)
	if err != nil {
		return "", "", nil, fmt.Errorf("generate qr: %w", err)
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return "", "", nil, fmt.Errorf("encode qr png: %w", err)
	}

	return key.Secret(), key.URL(), buf.Bytes(), nil
}

// Validate checks a 6-digit code against a base32 secret.
// Accepts codes from the previous, current, and next 30-second window
// to tolerate clock skew.
func Validate(code, secret string) bool {
	ok, err := totp.ValidateCustom(code, secret, time.Now().UTC(), totp.ValidateOpts{
		Period:    30,
		Skew:      1,
		Digits:    gotp.DigitsSix,
		Algorithm: gotp.AlgorithmSHA1,
	})
	return err == nil && ok
}

// GenerateRecoveryCodes creates 10 single-use recovery codes.
// Returns the plaintext codes (to show the user once) and their SHA-256 hashes
// (to store in the database).
// Each code is formatted as "xxxxxx-xxxxxx" (12 lowercase hex chars).
func GenerateRecoveryCodes() (plaintext []string, hashes []string, err error) {
	for i := 0; i < 10; i++ {
		b := make([]byte, 6)
		if _, err := rand.Read(b); err != nil {
			return nil, nil, fmt.Errorf("generate recovery code: %w", err)
		}
		code := hex.EncodeToString(b[:3]) + "-" + hex.EncodeToString(b[3:])
		plaintext = append(plaintext, code)
		hashes = append(hashes, hashCode(code))
	}
	return plaintext, hashes, nil
}

// HashRecoveryCode returns the canonical SHA-256 hash of a submitted recovery code.
// The code is lowercased before hashing so submission is case-insensitive.
func HashRecoveryCode(code string) string {
	return hashCode(strings.ToLower(strings.TrimSpace(code)))
}

func hashCode(code string) string {
	h := sha256.Sum256([]byte(code))
	return hex.EncodeToString(h[:])
}
