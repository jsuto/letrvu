package totp

import (
	"strings"
	"testing"
	"time"

	gototp "github.com/pquerna/otp/totp"
)

func TestGenerate(t *testing.T) {
	secret, url, qr, err := Generate("letrvu", "test@example.com")
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if secret == "" {
		t.Error("secret is empty")
	}
	if !strings.HasPrefix(url, "otpauth://totp/") {
		t.Errorf("unexpected url: %s", url)
	}
	if len(qr) == 0 {
		t.Error("qr png is empty")
	}
}

func TestValidate(t *testing.T) {
	secret, _, _, err := Generate("letrvu", "test@example.com")
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}

	code, err := gototp.GenerateCode(secret, time.Now())
	if err != nil {
		t.Fatalf("GenerateCode: %v", err)
	}

	if !Validate(code, secret) {
		t.Error("Validate returned false for a valid code")
	}
	if Validate("000000", secret) {
		t.Error("Validate returned true for a likely-invalid code")
	}
}

func TestGenerateRecoveryCodes(t *testing.T) {
	plain, hashes, err := GenerateRecoveryCodes()
	if err != nil {
		t.Fatalf("GenerateRecoveryCodes: %v", err)
	}
	if len(plain) != 10 || len(hashes) != 10 {
		t.Errorf("expected 10 codes, got plain=%d hashes=%d", len(plain), len(hashes))
	}

	// Each code should be in "xxxxxx-xxxxxx" format
	for _, c := range plain {
		parts := strings.Split(c, "-")
		if len(parts) != 2 || len(parts[0]) != 6 || len(parts[1]) != 6 {
			t.Errorf("unexpected code format: %q", c)
		}
	}

	// Hashes must be unique
	seen := map[string]bool{}
	for _, h := range hashes {
		if seen[h] {
			t.Error("duplicate hash")
		}
		seen[h] = true
	}
}

func TestHashRecoveryCode(t *testing.T) {
	plain, hashes, err := GenerateRecoveryCodes()
	if err != nil {
		t.Fatalf("GenerateRecoveryCodes: %v", err)
	}
	for i, code := range plain {
		if HashRecoveryCode(code) != hashes[i] {
			t.Errorf("hash mismatch for code %q", code)
		}
		// Case-insensitive
		if HashRecoveryCode(strings.ToUpper(code)) != hashes[i] {
			t.Errorf("hash not case-insensitive for code %q", code)
		}
	}
}
