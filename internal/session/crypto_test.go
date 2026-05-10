package session

import (
	"strings"
	"testing"
)

func TestDeriveKey_Length(t *testing.T) {
	key, err := deriveKey([]byte("server-secret-32-bytes-long-xxxx"), []byte("client-nonce"))
	if err != nil {
		t.Fatalf("deriveKey: %v", err)
	}
	if len(key) != 32 {
		t.Fatalf("want 32-byte key, got %d", len(key))
	}
}

func TestDeriveKey_Deterministic(t *testing.T) {
	secret := []byte("server-secret-32-bytes-long-xxxx")
	nonce := []byte("client-nonce")
	k1, _ := deriveKey(secret, nonce)
	k2, _ := deriveKey(secret, nonce)
	if string(k1) != string(k2) {
		t.Fatal("deriveKey should be deterministic for the same inputs")
	}
}

func TestDeriveKey_DifferentNonce(t *testing.T) {
	secret := []byte("server-secret-32-bytes-long-xxxx")
	k1, _ := deriveKey(secret, []byte("nonce-A"))
	k2, _ := deriveKey(secret, []byte("nonce-B"))
	if string(k1) == string(k2) {
		t.Fatal("different nonces should produce different keys")
	}
}

func TestDeriveKey_DifferentSecret(t *testing.T) {
	nonce := []byte("same-nonce")
	k1, _ := deriveKey([]byte("secret-one-32-bytes-long-xxxxxxx"), nonce)
	k2, _ := deriveKey([]byte("secret-two-32-bytes-long-xxxxxxx"), nonce)
	if string(k1) == string(k2) {
		t.Fatal("different secrets should produce different keys")
	}
}

func TestEncryptDecrypt_RoundTrip(t *testing.T) {
	secret := []byte("server-secret-32-bytes-long-xxxx")
	nonce := []byte("client-nonce")
	plaintext := "hunter2"

	enc, err := encryptPassword(secret, nonce, plaintext)
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}

	got, err := decryptPassword(secret, nonce, enc)
	if err != nil {
		t.Fatalf("decrypt: %v", err)
	}
	if got != plaintext {
		t.Fatalf("want %q, got %q", plaintext, got)
	}
}

func TestEncryptDecrypt_EmptyPassword(t *testing.T) {
	secret := []byte("server-secret-32-bytes-long-xxxx")
	nonce := []byte("nonce")

	enc, err := encryptPassword(secret, nonce, "")
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}
	got, err := decryptPassword(secret, nonce, enc)
	if err != nil {
		t.Fatalf("decrypt: %v", err)
	}
	if got != "" {
		t.Fatalf("want empty string, got %q", got)
	}
}

func TestDecrypt_WrongNonce(t *testing.T) {
	secret := []byte("server-secret-32-bytes-long-xxxx")
	enc, _ := encryptPassword(secret, []byte("nonce-A"), "secret")
	_, err := decryptPassword(secret, []byte("nonce-B"), enc)
	if err == nil {
		t.Fatal("expected error decrypting with wrong nonce")
	}
}

func TestDecrypt_WrongSecret(t *testing.T) {
	nonce := []byte("nonce")
	enc, _ := encryptPassword([]byte("secret-one-32-bytes-long-xxxxxxx"), nonce, "secret")
	_, err := decryptPassword([]byte("secret-two-32-bytes-long-xxxxxxx"), nonce, enc)
	if err == nil {
		t.Fatal("expected error decrypting with wrong secret")
	}
}

func TestDecrypt_Tampered(t *testing.T) {
	secret := []byte("server-secret-32-bytes-long-xxxx")
	nonce := []byte("nonce")
	enc, _ := encryptPassword(secret, nonce, "mypassword")

	// Flip the last character of the base64 ciphertext.
	b := []byte(enc)
	b[len(b)-1] ^= 0x01
	_, err := decryptPassword(secret, nonce, string(b))
	if err == nil {
		t.Fatal("expected error for tampered ciphertext")
	}
}

func TestEncrypt_NondeterministicOutput(t *testing.T) {
	// Each call should produce a different ciphertext (random GCM nonce).
	secret := []byte("server-secret-32-bytes-long-xxxx")
	nonce := []byte("nonce")
	e1, _ := encryptPassword(secret, nonce, "password")
	e2, _ := encryptPassword(secret, nonce, "password")
	if e1 == e2 {
		t.Fatal("two encryptions of the same plaintext should differ (random nonce)")
	}
}

func TestDecrypt_InvalidBase64(t *testing.T) {
	secret := []byte("server-secret-32-bytes-long-xxxx")
	_, err := decryptPassword(secret, []byte("nonce"), "not-valid-base64!!!")
	if err == nil {
		t.Fatal("expected error for invalid base64")
	}
}

func TestDecrypt_TooShort(t *testing.T) {
	secret := []byte("server-secret-32-bytes-long-xxxx")
	// Valid base64 but too short to contain a nonce.
	_, err := decryptPassword(secret, []byte("nonce"), "YWJj") // "abc"
	if err == nil {
		t.Fatal("expected error for ciphertext shorter than nonce")
	}
}

func TestEncryptDecrypt_LongPassword(t *testing.T) {
	secret := []byte("server-secret-32-bytes-long-xxxx")
	nonce := []byte("nonce")
	long := strings.Repeat("x", 10_000)

	enc, err := encryptPassword(secret, nonce, long)
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}
	got, err := decryptPassword(secret, nonce, enc)
	if err != nil {
		t.Fatalf("decrypt: %v", err)
	}
	if got != long {
		t.Fatalf("round-trip failed for long password")
	}
}
