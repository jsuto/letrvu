package calendar

import (
	"time"

	rrulelib "github.com/teambition/rrule-go"
)

// expandRRule returns the start times of all occurrences of a recurring event
// that overlap the window [from, to].
//
// rule is a bare RFC 5545 RRULE value (e.g. "FREQ=WEEKLY;BYDAY=MO,WE").
// baseStart is the event's original DTSTART; duration is ends_at - starts_at.
func expandRRule(rule string, baseStart time.Time, duration time.Duration, from, to time.Time) ([]time.Time, error) {
	opt, err := rrulelib.StrToROption(rule)
	if err != nil {
		return nil, err
	}
	opt.Dtstart = baseStart.UTC()

	r, err := rrulelib.NewRRule(*opt)
	if err != nil {
		return nil, err
	}

	// An occurrence overlaps [from, to] when its start < to AND its end > from.
	// So we ask for occurrences with start in (from-duration, to] and then
	// filter precisely to exclude occurrences that end exactly at from.
	searchFrom := from.Add(-duration)
	occStarts := r.Between(searchFrom, to, true)

	var result []time.Time
	for _, occ := range occStarts {
		occEnd := occ.Add(duration)
		if !occ.After(to) && occEnd.After(from) {
			result = append(result, occ)
		}
	}
	return result, nil
}
