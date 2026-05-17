//go:build integration

package api_test

import (
	"encoding/json"
	"io"
	"net/http"
	"testing"
)

func TestVacation_GetDefaults(t *testing.T) {
	env := newTestEnv(t)

	var resp struct {
		Enabled         bool   `json:"enabled"`
		Subject         string `json:"subject"`
		SieveConfigured bool   `json:"sieve_configured"`
		SieveActive     bool   `json:"sieve_active"`
	}
	env.getJSON(t, "/api/vacation", &resp)

	if resp.Enabled {
		t.Error("vacation should be disabled by default")
	}
	if resp.SieveConfigured {
		t.Error("sieve_configured should be false when SIEVE_HOST is not set")
	}
	if resp.SieveActive {
		t.Error("sieve_active should be false by default")
	}
	if resp.Subject != "" {
		t.Errorf("subject should be empty by default, got %q", resp.Subject)
	}
}

func TestVacation_SetAndGet(t *testing.T) {
	env := newTestEnv(t)

	// Save vacation settings (no SIEVE_HOST in testEnv → sieve_configured=false).
	body := `{"enabled":true,"subject":"Out of office","body":"I am away.","start":"","end":""}`
	resp := env.do(t, http.MethodPut, "/api/vacation", body)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("PUT /api/vacation: want 200, got %d: %s", resp.StatusCode, b)
	}

	var put struct {
		Enabled         bool   `json:"enabled"`
		Subject         string `json:"subject"`
		Body            string `json:"body"`
		SieveConfigured bool   `json:"sieve_configured"`
		SieveActive     bool   `json:"sieve_active"`
	}
	json.NewDecoder(resp.Body).Decode(&put) //nolint:errcheck

	if !put.Enabled {
		t.Error("enabled should be true")
	}
	if put.Subject != "Out of office" {
		t.Errorf("subject: want %q, got %q", "Out of office", put.Subject)
	}
	if put.SieveConfigured {
		t.Error("sieve_configured should be false when SIEVE_HOST is not set in testEnv")
	}
	if put.SieveActive {
		t.Error("sieve_active should be false without a real ManageSieve server")
	}

	// GET must reflect what was saved.
	var get struct {
		Enabled bool   `json:"enabled"`
		Subject string `json:"subject"`
		Body    string `json:"body"`
	}
	env.getJSON(t, "/api/vacation", &get)

	if !get.Enabled {
		t.Error("GET: enabled should be true after save")
	}
	if get.Subject != "Out of office" {
		t.Errorf("GET: subject: want %q, got %q", "Out of office", get.Subject)
	}
	if get.Body != "I am away." {
		t.Errorf("GET: body: want %q, got %q", "I am away.", get.Body)
	}
}

func TestVacation_SetWithDateRange(t *testing.T) {
	env := newTestEnv(t)

	body := `{"enabled":true,"subject":"On leave","body":"Back on the 16th.","start":"2026-06-01","end":"2026-06-15"}`
	resp := env.do(t, http.MethodPut, "/api/vacation", body)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("PUT /api/vacation: want 200, got %d: %s", resp.StatusCode, b)
	}

	var get struct {
		Start string `json:"start"`
		End   string `json:"end"`
	}
	env.getJSON(t, "/api/vacation", &get)

	if get.Start != "2026-06-01" {
		t.Errorf("start: want %q, got %q", "2026-06-01", get.Start)
	}
	if get.End != "2026-06-15" {
		t.Errorf("end: want %q, got %q", "2026-06-15", get.End)
	}
}

func TestVacation_DisableClears(t *testing.T) {
	env := newTestEnv(t)

	// Enable first.
	env.do(t, http.MethodPut, "/api/vacation",
		`{"enabled":true,"subject":"OOO","body":"Away."}`).Body.Close()

	// Now disable.
	resp := env.do(t, http.MethodPut, "/api/vacation", `{"enabled":false,"subject":"","body":""}`)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("disable: want 200, got %d: %s", resp.StatusCode, b)
	}

	var get struct {
		Enabled bool `json:"enabled"`
	}
	env.getJSON(t, "/api/vacation", &get)
	if get.Enabled {
		t.Error("vacation should be disabled after PUT enabled=false")
	}
}

func TestVacation_ValidationMissingSubject(t *testing.T) {
	env := newTestEnv(t)

	resp := env.do(t, http.MethodPut, "/api/vacation", `{"enabled":true,"subject":"","body":"Away."}`)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("missing subject: want 400, got %d", resp.StatusCode)
	}
}

func TestVacation_ValidationMissingBody(t *testing.T) {
	env := newTestEnv(t)

	resp := env.do(t, http.MethodPut, "/api/vacation", `{"enabled":true,"subject":"OOO","body":""}`)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("missing body: want 400, got %d", resp.StatusCode)
	}
}

func TestVacation_ValidationNotRequiredWhenDisabled(t *testing.T) {
	env := newTestEnv(t)

	// Disabling with empty subject/body should be fine.
	resp := env.do(t, http.MethodPut, "/api/vacation", `{"enabled":false,"subject":"","body":""}`)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("disable with empty fields: want 200, got %d: %s", resp.StatusCode, b)
	}
}

func TestVacation_InvalidJSON(t *testing.T) {
	env := newTestEnv(t)

	resp := env.do(t, http.MethodPut, "/api/vacation", `not json`)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("invalid JSON: want 400, got %d", resp.StatusCode)
	}
}
