package session

import (
	"testing"
	"time"

	"github.com/jsuto/letrvu/internal/db"
)

var testSecret = []byte("server-secret-32-bytes-long-xxxx")

func newTestStore(t *testing.T, idleTimeout time.Duration) *Store {
	t.Helper()
	database, err := db.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("db.Open: %v", err)
	}
	if err := db.Migrate(database); err != nil {
		t.Fatalf("db.Migrate: %v", err)
	}
	t.Cleanup(func() { database.Close() })
	return NewStore(database, testSecret, idleTimeout)
}

func TestStore_CreateAndGet(t *testing.T) {
	s := newTestStore(t, 0)
	cookie, err := s.Create("imap.example.com", 993, "smtp.example.com", 587, "user@example.com", "hunter2", "TestBrowser/1.0")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	sess, ok := s.Get(cookie)
	if !ok {
		t.Fatal("Get returned false for a fresh session")
	}
	if sess.Username != "user@example.com" {
		t.Errorf("Username = %q, want %q", sess.Username, "user@example.com")
	}
	if sess.Password != "hunter2" {
		t.Errorf("Password not decrypted correctly")
	}
	if sess.UserAgent != "TestBrowser/1.0" {
		t.Errorf("UserAgent = %q, want %q", sess.UserAgent, "TestBrowser/1.0")
	}
	if sess.LastActivityAt.IsZero() {
		t.Error("LastActivityAt should be set after Create")
	}
}

func TestStore_Get_InvalidCookie(t *testing.T) {
	s := newTestStore(t, 0)
	if _, ok := s.Get("notvalid"); ok {
		t.Error("expected false for malformed cookie")
	}
	if _, ok := s.Get("missing.nonce"); ok {
		t.Error("expected false for unknown session id")
	}
}

func TestStore_Delete(t *testing.T) {
	s := newTestStore(t, 0)
	cookie, _ := s.Create("imap.example.com", 993, "smtp.example.com", 587, "u@e.com", "pw", "UA")
	s.Delete(cookie)
	if _, ok := s.Get(cookie); ok {
		t.Error("Get should return false after Delete")
	}
}

func TestStore_DeleteExpired(t *testing.T) {
	s := newTestStore(t, 0)
	// Manually insert an already-expired session.
	past := time.Now().UTC().Add(-time.Hour).Format(time.RFC3339)
	s.db.Exec(s.db.Q(`INSERT INTO sessions
		(id, username, imap_host, imap_port, smtp_host, smtp_port, encrypted_password, created_at, expires_at, user_agent, last_activity_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`),
		"expiredid", "u@e.com", "imap.example.com", 993, "smtp.example.com", 587, "enc",
		past, past, "UA", past,
	)
	s.DeleteExpired()
	var count int
	s.db.QueryRow(`SELECT COUNT(*) FROM sessions WHERE id = 'expiredid'`).Scan(&count)
	if count != 0 {
		t.Error("DeleteExpired should have removed the expired session")
	}
}

func TestStore_IdleTimeout_Enforced(t *testing.T) {
	s := newTestStore(t, 30*time.Minute)

	cookie, _ := s.Create("imap.example.com", 993, "smtp.example.com", 587, "u@e.com", "pw", "UA")

	// Backdate last_activity_at to beyond the idle timeout.
	id, _, _ := parseCookieValue(cookie)
	old := time.Now().UTC().Add(-time.Hour).Format(time.RFC3339)
	s.db.Exec(s.db.Q(`UPDATE sessions SET last_activity_at = ? WHERE id = ?`), old, id)

	if _, ok := s.Get(cookie); ok {
		t.Error("Get should return false after idle timeout exceeded")
	}
}

func TestStore_IdleTimeout_Disabled(t *testing.T) {
	s := newTestStore(t, 0) // idle timeout disabled

	cookie, _ := s.Create("imap.example.com", 993, "smtp.example.com", 587, "u@e.com", "pw", "UA")

	// Backdate last_activity_at — should not matter when timeout is 0.
	id, _, _ := parseCookieValue(cookie)
	old := time.Now().UTC().Add(-365 * 24 * time.Hour).Format(time.RFC3339)
	s.db.Exec(s.db.Q(`UPDATE sessions SET last_activity_at = ? WHERE id = ?`), old, id)

	if _, ok := s.Get(cookie); !ok {
		t.Error("Get should succeed when idle timeout is disabled")
	}
}

func TestStore_List(t *testing.T) {
	s := newTestStore(t, 0)

	cookie1, _ := s.Create("imap.example.com", 993, "smtp.example.com", 587, "u@e.com", "pw", "Firefox/120")
	cookie2, _ := s.Create("imap.example.com", 993, "smtp.example.com", 587, "u@e.com", "pw", "Chrome/120")
	// Different user — should not appear.
	s.Create("imap.example.com", 993, "smtp.example.com", 587, "other@e.com", "pw", "Safari")

	id1, _, _ := parseCookieValue(cookie1)
	sessions, err := s.List("u@e.com", "imap.example.com", id1)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(sessions) != 2 {
		t.Fatalf("expected 2 sessions, got %d", len(sessions))
	}

	var foundCurrent bool
	for _, sess := range sessions {
		if sess.ID == id1 {
			if !sess.Current {
				t.Error("session 1 should be marked Current")
			}
			foundCurrent = true
		}
	}
	if !foundCurrent {
		t.Error("current session not found in list")
	}
	_ = cookie2
}

func TestStore_DeleteAllForUser_ExceptCurrent(t *testing.T) {
	s := newTestStore(t, 0)

	cookie1, _ := s.Create("imap.example.com", 993, "smtp.example.com", 587, "u@e.com", "pw", "UA1")
	cookie2, _ := s.Create("imap.example.com", 993, "smtp.example.com", 587, "u@e.com", "pw", "UA2")
	// Different user — must not be touched.
	cookieOther, _ := s.Create("imap.example.com", 993, "smtp.example.com", 587, "other@e.com", "pw", "UA3")

	id1, _, _ := parseCookieValue(cookie1)
	s.DeleteAllForUser("u@e.com", "imap.example.com", id1)

	if _, ok := s.Get(cookie1); !ok {
		t.Error("current session (cookie1) should still be valid")
	}
	if _, ok := s.Get(cookie2); ok {
		t.Error("other session (cookie2) should have been deleted")
	}
	if _, ok := s.Get(cookieOther); !ok {
		t.Error("other user's session should not be affected")
	}
}

func TestStore_DeleteAllForUser_IncludingCurrent(t *testing.T) {
	s := newTestStore(t, 0)

	cookie1, _ := s.Create("imap.example.com", 993, "smtp.example.com", 587, "u@e.com", "pw", "UA1")
	cookie2, _ := s.Create("imap.example.com", 993, "smtp.example.com", 587, "u@e.com", "pw", "UA2")

	s.DeleteAllForUser("u@e.com", "imap.example.com", "")

	if _, ok := s.Get(cookie1); ok {
		t.Error("cookie1 should have been deleted")
	}
	if _, ok := s.Get(cookie2); ok {
		t.Error("cookie2 should have been deleted")
	}
}
