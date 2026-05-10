package session

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"

	"golang.org/x/crypto/hkdf"
)

// deriveKey derives a 32-byte AES-256 key from the server secret and the
// per-session client nonce using HKDF-SHA256. Neither the secret alone nor
// the nonce alone is sufficient to reproduce the key.
func deriveKey(serverSecret, clientNonce []byte) ([]byte, error) {
	r := hkdf.New(sha256.New, serverSecret, clientNonce, []byte("letrvu-imap-key"))
	key := make([]byte, 32)
	if _, err := io.ReadFull(r, key); err != nil {
		return nil, err
	}
	return key, nil
}

// encryptPassword encrypts password with AES-256-GCM and returns a
// base64-encoded ciphertext (nonce prepended).
func encryptPassword(serverSecret, clientNonce []byte, password string) (string, error) {
	key, err := deriveKey(serverSecret, clientNonce)
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
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", err
	}
	ciphertext := gcm.Seal(nonce, nonce, []byte(password), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// decryptPassword decrypts a base64-encoded AES-256-GCM ciphertext.
func decryptPassword(serverSecret, clientNonce []byte, encrypted string) (string, error) {
	key, err := deriveKey(serverSecret, clientNonce)
	if err != nil {
		return "", err
	}
	ciphertext, err := base64.StdEncoding.DecodeString(encrypted)
	if err != nil {
		return "", fmt.Errorf("decode: %w", err)
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
