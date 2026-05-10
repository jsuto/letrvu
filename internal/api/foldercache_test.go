package api

import (
	"testing"
	"time"

	"github.com/jsuto/letrvu/internal/imap"
)

var testFolders = []imap.Folder{
	{Name: "INBOX", Unseen: 3},
	{Name: "Sent"},
}

func TestFolderCache_MissOnEmpty(t *testing.T) {
	fc := newFolderCache(time.Minute)
	_, ok := fc.get("user@host")
	if ok {
		t.Error("empty cache should be a miss")
	}
}

func TestFolderCache_HitAfterSet(t *testing.T) {
	fc := newFolderCache(time.Minute)
	fc.set("user@host", testFolders)
	got, ok := fc.get("user@host")
	if !ok {
		t.Fatal("expected cache hit")
	}
	if len(got) != len(testFolders) {
		t.Errorf("got %d folders, want %d", len(got), len(testFolders))
	}
}

func TestFolderCache_MissDifferentKey(t *testing.T) {
	fc := newFolderCache(time.Minute)
	fc.set("user@host-a", testFolders)
	_, ok := fc.get("user@host-b")
	if ok {
		t.Error("different key should be a miss")
	}
}

func TestFolderCache_NotStaleAfterSet(t *testing.T) {
	fc := newFolderCache(time.Minute)
	fc.set("user@host", testFolders)
	if fc.stale("user@host") {
		t.Error("freshly set entry should not be stale")
	}
}

func TestFolderCache_StaleWithZeroTTL(t *testing.T) {
	fc := newFolderCache(0)
	fc.set("user@host", testFolders)
	// With TTL=0, any entry is immediately stale.
	if !fc.stale("user@host") {
		t.Error("entry should be stale with TTL=0")
	}
}

func TestFolderCache_StaleOnMiss(t *testing.T) {
	fc := newFolderCache(time.Minute)
	if !fc.stale("nonexistent") {
		t.Error("missing entry should be considered stale")
	}
}

func TestFolderCache_Invalidate(t *testing.T) {
	fc := newFolderCache(time.Minute)
	fc.set("user@host", testFolders)
	fc.invalidate("user@host")
	_, ok := fc.get("user@host")
	if ok {
		t.Error("invalidated entry should be a miss")
	}
}

func TestFolderCache_InvalidateNonexistent(t *testing.T) {
	// Should not panic.
	fc := newFolderCache(time.Minute)
	fc.invalidate("nobody@nowhere")
}

func TestFolderCache_MarkRefreshing(t *testing.T) {
	fc := newFolderCache(time.Minute)
	key := "user@host"

	// First call should succeed.
	if !fc.markRefreshing(key) {
		t.Error("first markRefreshing should return true")
	}
	// Second call while refresh is in-flight should return false.
	if fc.markRefreshing(key) {
		t.Error("second markRefreshing should return false (already in-flight)")
	}
}

func TestFolderCache_RefreshingResetOnSet(t *testing.T) {
	fc := newFolderCache(time.Minute)
	key := "user@host"

	fc.markRefreshing(key)
	fc.set(key, testFolders) // set clears the refreshing flag

	// Should be able to mark refreshing again.
	if !fc.markRefreshing(key) {
		t.Error("markRefreshing should return true after set() clears the flag")
	}
}

func TestCacheKey(t *testing.T) {
	got := cacheKey("alice", "imap.example.com")
	want := "alice@imap.example.com"
	if got != want {
		t.Errorf("cacheKey = %q, want %q", got, want)
	}
}
