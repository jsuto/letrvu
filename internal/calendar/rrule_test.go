package calendar

import (
	"testing"
	"time"
)

var (
	baseTime = time.Date(2024, 3, 4, 9, 0, 0, 0, time.UTC) // Monday
	oneHour  = time.Hour
)

func mustExpand(t *testing.T, rule string, base time.Time, dur time.Duration, from, to time.Time) []time.Time {
	t.Helper()
	starts, err := expandRRule(rule, base, dur, from, to)
	if err != nil {
		t.Fatalf("expandRRule(%q): %v", rule, err)
	}
	return starts
}

// --- DAILY -------------------------------------------------------------------

func TestRRule_Daily_Basic(t *testing.T) {
	// Every day for a week, query for 3 days.
	from := baseTime
	to := baseTime.AddDate(0, 0, 2) // Mon-Wed
	starts := mustExpand(t, "FREQ=DAILY", baseTime, oneHour, from, to)
	if len(starts) != 3 {
		t.Fatalf("want 3 occurrences, got %d: %v", len(starts), starts)
	}
	for i, s := range starts {
		want := baseTime.AddDate(0, 0, i)
		if !s.Equal(want) {
			t.Errorf("starts[%d] = %v, want %v", i, s, want)
		}
	}
}

func TestRRule_Daily_Interval(t *testing.T) {
	// Every 2 days.
	from := baseTime
	to := baseTime.AddDate(0, 0, 6)
	starts := mustExpand(t, "FREQ=DAILY;INTERVAL=2", baseTime, oneHour, from, to)
	// Occurrences: day 0, 2, 4, 6 → 4 total.
	if len(starts) != 4 {
		t.Fatalf("want 4, got %d: %v", len(starts), starts)
	}
}

func TestRRule_Daily_Count(t *testing.T) {
	from := baseTime
	to := baseTime.AddDate(0, 1, 0)
	starts := mustExpand(t, "FREQ=DAILY;COUNT=5", baseTime, oneHour, from, to)
	if len(starts) != 5 {
		t.Fatalf("want 5, got %d", len(starts))
	}
}

func TestRRule_Daily_Until(t *testing.T) {
	until := baseTime.AddDate(0, 0, 3) // 4 days inclusive
	rule := "FREQ=DAILY;UNTIL=" + until.Format("20060102T150405Z")
	from := baseTime
	to := baseTime.AddDate(0, 1, 0)
	starts := mustExpand(t, rule, baseTime, oneHour, from, to)
	if len(starts) != 4 {
		t.Fatalf("want 4 (day 0-3 inclusive), got %d: %v", len(starts), starts)
	}
}

func TestRRule_Daily_WindowBeforeBase(t *testing.T) {
	// Query window is entirely before the base start.
	from := baseTime.AddDate(0, 0, -10)
	to := baseTime.AddDate(0, 0, -1)
	starts := mustExpand(t, "FREQ=DAILY", baseTime, oneHour, from, to)
	if len(starts) != 0 {
		t.Errorf("expected 0 occurrences before base, got %d", len(starts))
	}
}

func TestRRule_Daily_WindowAfterBase(t *testing.T) {
	// Query window starts 30 days after base — recurrence should still generate.
	from := baseTime.AddDate(0, 0, 30)
	to := baseTime.AddDate(0, 0, 32)
	starts := mustExpand(t, "FREQ=DAILY", baseTime, oneHour, from, to)
	if len(starts) != 3 {
		t.Fatalf("want 3 occurrences in future window, got %d", len(starts))
	}
}

// --- WEEKLY ------------------------------------------------------------------

func TestRRule_Weekly_Basic(t *testing.T) {
	// Every Monday starting 2024-03-04, query a month.
	from := baseTime
	to := baseTime.AddDate(0, 1, 0)
	starts := mustExpand(t, "FREQ=WEEKLY", baseTime, oneHour, from, to)
	// ~4-5 Mondays in a month.
	if len(starts) < 4 {
		t.Fatalf("want at least 4 weekly occurrences, got %d", len(starts))
	}
	// All should be Mondays.
	for _, s := range starts {
		if s.Weekday() != time.Monday {
			t.Errorf("expected Monday, got %v", s.Weekday())
		}
	}
}

func TestRRule_Weekly_BYDAY(t *testing.T) {
	// Mon/Wed/Fri for one week.
	from := baseTime // Monday
	to := baseTime.AddDate(0, 0, 6)
	starts := mustExpand(t, "FREQ=WEEKLY;BYDAY=MO,WE,FR", baseTime, oneHour, from, to)
	if len(starts) != 3 {
		t.Fatalf("want 3 (Mon/Wed/Fri), got %d: %v", len(starts), starts)
	}
	days := []time.Weekday{time.Monday, time.Wednesday, time.Friday}
	for i, s := range starts {
		if s.Weekday() != days[i] {
			t.Errorf("starts[%d].Weekday() = %v, want %v", i, s.Weekday(), days[i])
		}
	}
}

func TestRRule_Weekly_BYDAY_TwoWeeks(t *testing.T) {
	// Tue/Thu for 2 weeks.
	from := baseTime // Monday
	to := baseTime.AddDate(0, 0, 13)
	starts := mustExpand(t, "FREQ=WEEKLY;BYDAY=TU,TH", baseTime, oneHour, from, to)
	if len(starts) != 4 {
		t.Fatalf("want 4 (Tue+Thu × 2 weeks), got %d: %v", len(starts), starts)
	}
}

func TestRRule_Weekly_Interval2(t *testing.T) {
	// Every 2 weeks for 4 weeks.
	from := baseTime
	to := baseTime.AddDate(0, 0, 28)
	starts := mustExpand(t, "FREQ=WEEKLY;INTERVAL=2", baseTime, oneHour, from, to)
	if len(starts) != 3 { // weeks 0, 2, 4 but 4 weeks → days 0, 14, 28
		t.Fatalf("want 3, got %d: %v", len(starts), starts)
	}
}

// --- MONTHLY -----------------------------------------------------------------

func TestRRule_Monthly_Basic(t *testing.T) {
	from := baseTime
	to := baseTime.AddDate(0, 3, 0) // 3 months
	starts := mustExpand(t, "FREQ=MONTHLY", baseTime, oneHour, from, to)
	if len(starts) != 4 { // month 0, 1, 2, 3 (inclusive since to is at start of month 3+1)
		t.Fatalf("want 4, got %d: %v", len(starts), starts)
	}
}

func TestRRule_Monthly_Interval3(t *testing.T) {
	from := baseTime
	to := baseTime.AddDate(1, 0, 0) // 12 months
	starts := mustExpand(t, "FREQ=MONTHLY;INTERVAL=3", baseTime, oneHour, from, to)
	if len(starts) != 5 { // months 0, 3, 6, 9, 12
		t.Fatalf("want 5, got %d: %v", len(starts), starts)
	}
}

// --- YEARLY ------------------------------------------------------------------

func TestRRule_Yearly_Basic(t *testing.T) {
	from := baseTime
	to := baseTime.AddDate(3, 0, 0)
	starts := mustExpand(t, "FREQ=YEARLY", baseTime, oneHour, from, to)
	if len(starts) != 4 { // years 0,1,2,3
		t.Fatalf("want 4, got %d: %v", len(starts), starts)
	}
}

// --- Edge cases --------------------------------------------------------------

func TestRRule_InvalidRule(t *testing.T) {
	_, err := expandRRule("NOT_VALID_RULE", baseTime, oneHour, baseTime, baseTime.AddDate(0, 1, 0))
	if err == nil {
		t.Error("expected error for invalid RRULE")
	}
}

func TestRRule_EmptyRule(t *testing.T) {
	starts, err := expandRRule("", baseTime, oneHour, baseTime, baseTime.AddDate(0, 1, 0))
	// Empty rule errors in rrule-go; we treat it as no recurrence (store.go skips empty rrule)
	_ = err
	if len(starts) > 0 {
		t.Errorf("empty rule should produce no starts, got %d", len(starts))
	}
}

func TestRRule_OccurrenceOverlapsWindowStart(t *testing.T) {
	// 2-hour duration event. Base starts at 08:00. Window starts at 09:00.
	// The 08:00 occurrence ends at 10:00, so it overlaps the window [09:00, 18:00].
	base := time.Date(2024, 3, 4, 8, 0, 0, 0, time.UTC)
	dur := 2 * time.Hour
	from := time.Date(2024, 3, 4, 9, 0, 0, 0, time.UTC)
	to := time.Date(2024, 3, 4, 18, 0, 0, 0, time.UTC)
	starts := mustExpand(t, "FREQ=DAILY", base, dur, from, to)
	// The occurrence at 08:00 starts before from but ends after from — should be included.
	found := false
	for _, s := range starts {
		if s.Hour() == 8 {
			found = true
		}
	}
	if !found {
		t.Error("expected occurrence at 08:00 to be included when it overlaps window start")
	}
}
