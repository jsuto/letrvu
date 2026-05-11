package api

import (
	"sync"
	"time"
)

// loginLimiter tracks failed login attempts per IP address and enforces a
// temporary lockout after too many failures within a sliding window.
//
// All state is in-process memory; it resets on server restart. This is
// sufficient for protecting a personal webmail instance — a more durable
// solution (e.g. Redis or DB-backed) would be needed for multi-replica
// deployments.
type loginLimiter struct {
	mu       sync.Mutex
	entries  map[string]*limitEntry
	maxFails int
	window   time.Duration
	lockout  time.Duration
}

type limitEntry struct {
	failures    int
	windowStart time.Time
	lockedUntil time.Time
}

func newLoginLimiter(maxFails int, window, lockout time.Duration) *loginLimiter {
	l := &loginLimiter{
		entries:  make(map[string]*limitEntry),
		maxFails: maxFails,
		window:   window,
		lockout:  lockout,
	}
	go l.cleanup()
	return l
}

// blocked returns true and the remaining lockout duration if the IP is
// currently locked out.
func (l *loginLimiter) blocked(ip string) (bool, time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()
	e, ok := l.entries[ip]
	if !ok {
		return false, 0
	}
	if remaining := time.Until(e.lockedUntil); remaining > 0 {
		return true, remaining
	}
	return false, 0
}

// recordFailure increments the failure count for ip within the current window.
// Returns true if the IP has just been locked out.
func (l *loginLimiter) recordFailure(ip string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := time.Now()
	e, ok := l.entries[ip]
	if !ok {
		e = &limitEntry{windowStart: now}
		l.entries[ip] = e
	}
	// Reset counter when the window has expired.
	if now.Sub(e.windowStart) > l.window {
		e.failures = 0
		e.windowStart = now
	}
	e.failures++
	if e.failures >= l.maxFails {
		e.lockedUntil = now.Add(l.lockout)
		return true
	}
	return false
}

// recordSuccess clears the failure record for ip.
func (l *loginLimiter) recordSuccess(ip string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.entries, ip)
}

// cleanup removes stale entries periodically to prevent unbounded growth.
func (l *loginLimiter) cleanup() {
	ticker := time.NewTicker(l.lockout)
	defer ticker.Stop()
	for range ticker.C {
		l.mu.Lock()
		now := time.Now()
		for ip, e := range l.entries {
			if now.After(e.lockedUntil) && now.Sub(e.windowStart) > l.window {
				delete(l.entries, ip)
			}
		}
		l.mu.Unlock()
	}
}
