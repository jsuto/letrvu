package api

import (
	"testing"
	"time"
)

const testIP = "1.2.3.4"

func newTestLimiter(maxFails int) *loginLimiter {
	// Short window and lockout so tests don't have to sleep.
	return &loginLimiter{
		entries:  make(map[string]*limitEntry),
		maxFails: maxFails,
		window:   5 * time.Second,
		lockout:  10 * time.Second,
	}
}

// --- blocked -----------------------------------------------------------------

func TestLimiter_NotBlockedInitially(t *testing.T) {
	l := newTestLimiter(3)
	if blocked, _ := l.blocked(testIP); blocked {
		t.Error("fresh IP should not be blocked")
	}
}

// --- recordFailure -----------------------------------------------------------

func TestLimiter_BlockedAfterMaxFailures(t *testing.T) {
	l := newTestLimiter(3)
	for i := 0; i < 2; i++ {
		if locked := l.recordFailure(testIP); locked {
			t.Errorf("should not be locked after %d failures", i+1)
		}
		if blocked, _ := l.blocked(testIP); blocked {
			t.Errorf("should not be blocked after %d failures", i+1)
		}
	}
	// Third failure triggers lockout.
	if locked := l.recordFailure(testIP); !locked {
		t.Error("third failure should trigger lockout")
	}
	if blocked, _ := l.blocked(testIP); !blocked {
		t.Error("IP should be blocked after max failures")
	}
}

func TestLimiter_RemainingTimePositive(t *testing.T) {
	l := newTestLimiter(1)
	l.recordFailure(testIP)
	_, remaining := l.blocked(testIP)
	if remaining <= 0 {
		t.Errorf("remaining lockout should be positive, got %v", remaining)
	}
}

func TestLimiter_WindowResetAfterExpiry(t *testing.T) {
	l := &loginLimiter{
		entries:  make(map[string]*limitEntry),
		maxFails: 3,
		window:   50 * time.Millisecond,
		lockout:  10 * time.Second,
	}
	// Two failures within the window — not enough to lock.
	l.recordFailure(testIP)
	l.recordFailure(testIP)

	// Wait for the window to expire.
	time.Sleep(60 * time.Millisecond)

	// One failure in the new window — counter resets, so still not locked.
	locked := l.recordFailure(testIP)
	if locked {
		t.Error("failure after window expiry should reset the counter")
	}
	if blocked, _ := l.blocked(testIP); blocked {
		t.Error("IP should not be blocked after window reset")
	}
}

func TestLimiter_DifferentIPsAreIndependent(t *testing.T) {
	l := newTestLimiter(2)
	l.recordFailure("10.0.0.1")
	l.recordFailure("10.0.0.1")
	// 10.0.0.1 is now locked; 10.0.0.2 should be unaffected.
	if blocked, _ := l.blocked("10.0.0.2"); blocked {
		t.Error("unrelated IP should not be blocked")
	}
}

// --- recordSuccess -----------------------------------------------------------

func TestLimiter_SuccessClearsFailures(t *testing.T) {
	l := newTestLimiter(5)
	l.recordFailure(testIP)
	l.recordFailure(testIP)
	l.recordSuccess(testIP)
	// Failures should be cleared; two more failures should not lock.
	l.recordFailure(testIP)
	l.recordFailure(testIP)
	if blocked, _ := l.blocked(testIP); blocked {
		t.Error("IP should not be blocked after success cleared the counter")
	}
}

func TestLimiter_SuccessOnUnknownIPIsNoop(t *testing.T) {
	l := newTestLimiter(3)
	// Should not panic on an IP with no entry.
	l.recordSuccess("9.9.9.9")
	if blocked, _ := l.blocked("9.9.9.9"); blocked {
		t.Error("unknown IP should not be blocked")
	}
}

// --- cleanup -----------------------------------------------------------------

func TestLimiter_CleanupRemovesStaleEntries(t *testing.T) {
	l := &loginLimiter{
		entries:  make(map[string]*limitEntry),
		maxFails: 1,
		window:   1 * time.Millisecond,
		lockout:  1 * time.Millisecond,
	}
	l.recordFailure(testIP) // creates entry

	time.Sleep(5 * time.Millisecond)

	// Manually invoke cleanup logic (without the goroutine).
	l.mu.Lock()
	now := time.Now()
	for ip, e := range l.entries {
		if now.After(e.lockedUntil) && now.Sub(e.windowStart) > l.window {
			delete(l.entries, ip)
		}
	}
	l.mu.Unlock()

	if blocked, _ := l.blocked(testIP); blocked {
		t.Error("stale entry should have been cleaned up")
	}
}
