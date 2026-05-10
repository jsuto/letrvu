package settings

import (
	"testing"

	"github.com/jsuto/letrvu/internal/db"
)

// openTestDB creates an in-memory SQLite database with the user_settings table.
func openTestDB(t *testing.T) *db.DB {
	t.Helper()
	database, err := db.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if _, err := database.Exec(`CREATE TABLE user_settings (
		username  TEXT NOT NULL,
		imap_host TEXT NOT NULL,
		key       TEXT NOT NULL,
		value     TEXT NOT NULL,
		PRIMARY KEY (username, imap_host, key)
	)`); err != nil {
		t.Fatalf("create table: %v", err)
	}
	t.Cleanup(func() { database.Close() })
	return database
}

func newStore(t *testing.T) *Store {
	t.Helper()
	return NewStore(openTestDB(t))
}

// --- Get / Set round-trip ----------------------------------------------------

func TestSettings_SetAndGet(t *testing.T) {
	s := newStore(t)
	err := s.Set("alice", "imap.example.com", map[string]string{
		"display_name": "Alice",
		"signature":    "Best, Alice",
	})
	if err != nil {
		t.Fatalf("Set: %v", err)
	}

	got, err := s.Get("alice", "imap.example.com")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got["display_name"] != "Alice" {
		t.Errorf("display_name = %q, want %q", got["display_name"], "Alice")
	}
	if got["signature"] != "Best, Alice" {
		t.Errorf("signature = %q, want %q", got["signature"], "Best, Alice")
	}
}

func TestSettings_GetEmpty(t *testing.T) {
	s := newStore(t)
	got, err := s.Get("nobody", "imap.example.com")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("want empty map for unknown user, got %v", got)
	}
}

// --- Allowed key filter ------------------------------------------------------

func TestSettings_UnknownKeyIgnored(t *testing.T) {
	s := newStore(t)
	err := s.Set("alice", "imap.example.com", map[string]string{
		"display_name": "Alice",
		"evil_key":     "should be dropped",
	})
	if err != nil {
		t.Fatalf("Set: %v", err)
	}
	got, err := s.Get("alice", "imap.example.com")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if _, ok := got["evil_key"]; ok {
		t.Error("unknown key should not be persisted")
	}
	if got["display_name"] != "Alice" {
		t.Errorf("display_name = %q, want %q", got["display_name"], "Alice")
	}
}

func TestSettings_AllUnknownKeysNoError(t *testing.T) {
	s := newStore(t)
	// Should silently ignore all keys and succeed.
	if err := s.Set("alice", "imap.example.com", map[string]string{
		"unknown_a": "x",
		"unknown_b": "y",
	}); err != nil {
		t.Fatalf("Set with all-unknown keys should not error: %v", err)
	}
}

func TestSettings_AllAllowedKeys(t *testing.T) {
	s := newStore(t)
	values := map[string]string{
		"display_name": "Alice",
		"signature":    "sig",
		"identities":   `[{"name":"Alice","email":"a@example.com"}]`,
	}
	if err := s.Set("alice", "imap.example.com", values); err != nil {
		t.Fatalf("Set: %v", err)
	}
	got, err := s.Get("alice", "imap.example.com")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	for k, want := range values {
		if got[k] != want {
			t.Errorf("%s = %q, want %q", k, got[k], want)
		}
	}
}

// --- Upsert behaviour --------------------------------------------------------

func TestSettings_UpsertOverwrites(t *testing.T) {
	s := newStore(t)
	s.Set("alice", "imap.example.com", map[string]string{"display_name": "Alice"})
	s.Set("alice", "imap.example.com", map[string]string{"display_name": "Alice Smith"})

	got, _ := s.Get("alice", "imap.example.com")
	if got["display_name"] != "Alice Smith" {
		t.Errorf("display_name = %q, want %q", got["display_name"], "Alice Smith")
	}
}

// --- User isolation ----------------------------------------------------------

func TestSettings_UserIsolation(t *testing.T) {
	s := newStore(t)
	s.Set("alice", "imap.example.com", map[string]string{"display_name": "Alice"})
	s.Set("bob", "imap.example.com", map[string]string{"display_name": "Bob"})

	alice, _ := s.Get("alice", "imap.example.com")
	bob, _ := s.Get("bob", "imap.example.com")

	if alice["display_name"] != "Alice" {
		t.Errorf("alice display_name = %q", alice["display_name"])
	}
	if bob["display_name"] != "Bob" {
		t.Errorf("bob display_name = %q", bob["display_name"])
	}
}

func TestSettings_IMAPHostIsolation(t *testing.T) {
	s := newStore(t)
	s.Set("alice", "imap-a.example.com", map[string]string{"display_name": "Alice on A"})
	s.Set("alice", "imap-b.example.com", map[string]string{"display_name": "Alice on B"})

	a, _ := s.Get("alice", "imap-a.example.com")
	b, _ := s.Get("alice", "imap-b.example.com")

	if a["display_name"] != "Alice on A" {
		t.Errorf("imap-a display_name = %q", a["display_name"])
	}
	if b["display_name"] != "Alice on B" {
		t.Errorf("imap-b display_name = %q", b["display_name"])
	}
}
