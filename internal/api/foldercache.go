package api

import (
	"sync"
	"time"

	"github.com/jsuto/letrvu/internal/imap"
)

type folderCacheEntry struct {
	folders   []imap.Folder
	fetchedAt time.Time
}

// folderCache is a short-lived in-memory cache of folder lists keyed by
// "username@imapHost". It serves stale-while-revalidate: callers always get
// the last known list instantly; a background goroutine refreshes it.
type folderCache struct {
	ttl        time.Duration
	mu         sync.Mutex
	entries    map[string]*folderCacheEntry
	// refreshing tracks which keys have an in-flight background refresh so we
	// don't stack up multiple concurrent refreshes for the same user.
	refreshing map[string]bool
}

func newFolderCache(ttl time.Duration) *folderCache {
	return &folderCache{
		ttl:        ttl,
		entries:    make(map[string]*folderCacheEntry),
		refreshing: make(map[string]bool),
	}
}

func cacheKey(username, imapHost string) string {
	return username + "@" + imapHost
}

// get returns the cached folder list and whether the cache was a hit.
func (fc *folderCache) get(key string) ([]imap.Folder, bool) {
	fc.mu.Lock()
	defer fc.mu.Unlock()
	e, ok := fc.entries[key]
	if !ok {
		return nil, false
	}
	return e.folders, true
}

// set stores a fresh folder list.
func (fc *folderCache) set(key string, folders []imap.Folder) {
	fc.mu.Lock()
	defer fc.mu.Unlock()
	fc.entries[key] = &folderCacheEntry{folders: folders, fetchedAt: time.Now()}
	fc.refreshing[key] = false
}

// stale reports whether the cached entry is older than the TTL.
func (fc *folderCache) stale(key string) bool {
	fc.mu.Lock()
	defer fc.mu.Unlock()
	e, ok := fc.entries[key]
	return !ok || time.Since(e.fetchedAt) > fc.ttl
}

// markRefreshing records that a background refresh is in-flight.
// Returns false if one is already running (caller should skip launching another).
func (fc *folderCache) markRefreshing(key string) bool {
	fc.mu.Lock()
	defer fc.mu.Unlock()
	if fc.refreshing[key] {
		return false
	}
	fc.refreshing[key] = true
	return true
}

// invalidate removes a user's cache entry (e.g. on new-mail event).
func (fc *folderCache) invalidate(key string) {
	fc.mu.Lock()
	defer fc.mu.Unlock()
	delete(fc.entries, key)
}
