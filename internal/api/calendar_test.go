package api

import (
	"net/http/httptest"
	"testing"
	"time"
)

func TestMonthRange_WithBothParams(t *testing.T) {
	req := httptest.NewRequest("GET", "/?from=2024-03-01T00:00:00Z&to=2024-03-31T23:59:59Z", nil)
	from, to := monthRange(req)
	if from.Year() != 2024 || from.Month() != 3 || from.Day() != 1 {
		t.Errorf("from = %v, want 2024-03-01", from)
	}
	if to.Year() != 2024 || to.Month() != 3 || to.Day() != 31 {
		t.Errorf("to = %v, want 2024-03-31", to)
	}
}

func TestMonthRange_DefaultIsCurrentMonth(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	from, to := monthRange(req)
	now := time.Now()

	if from.Year() != now.Year() || from.Month() != now.Month() || from.Day() != 1 {
		t.Errorf("default from = %v, want first of current month", from)
	}
	expected := from.AddDate(0, 1, 0)
	if !to.Equal(expected) {
		t.Errorf("default to = %v, want %v", to, expected)
	}
}

func TestMonthRange_PartialParams_FallsBack(t *testing.T) {
	// Only 'from' — both params required, so falls back to current month.
	req := httptest.NewRequest("GET", "/?from=2024-03-01T00:00:00Z", nil)
	from, to := monthRange(req)
	now := time.Now()
	if from.Month() != now.Month() || from.Year() != now.Year() {
		t.Errorf("partial params should fall back to current month, got from=%v", from)
	}
	if !to.Equal(from.AddDate(0, 1, 0)) {
		t.Errorf("to should be one month after from, got %v", to)
	}
}

func TestMonthRange_InvalidParams_FallsBack(t *testing.T) {
	req := httptest.NewRequest("GET", "/?from=not-a-date&to=also-not-a-date", nil)
	from, to := monthRange(req)
	now := time.Now()
	if from.Month() != now.Month() || from.Year() != now.Year() {
		t.Errorf("invalid params should fall back to current month")
	}
	if !to.Equal(from.AddDate(0, 1, 0)) {
		t.Errorf("to should be one month after from")
	}
}

func TestMonthRange_DefaultStartsOnFirstDay(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	from, _ := monthRange(req)
	if from.Day() != 1 {
		t.Errorf("default from.Day() = %d, want 1", from.Day())
	}
	if from.Hour() != 0 || from.Minute() != 0 || from.Second() != 0 {
		t.Errorf("default from should be midnight, got %v", from)
	}
}
