package calendar

import (
	"strings"
	"testing"
	"time"
)

// --- parseICalTime -----------------------------------------------------------

func TestParseICalTime_UTC(t *testing.T) {
	got := parseICalTime("20240315T120000Z")
	want := time.Date(2024, 3, 15, 12, 0, 0, 0, time.UTC)
	if !got.Equal(want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestParseICalTime_Local(t *testing.T) {
	got := parseICalTime("20240315T120000")
	if got.IsZero() {
		t.Error("should parse local time without Z suffix")
	}
	if got.Hour() != 12 || got.Minute() != 0 {
		t.Errorf("got %v, want 12:00", got)
	}
}

func TestParseICalTime_DateOnly(t *testing.T) {
	got := parseICalTime("20240315")
	if got.IsZero() {
		t.Error("should parse date-only value")
	}
	if got.Year() != 2024 || got.Month() != 3 || got.Day() != 15 {
		t.Errorf("got %v, want 2024-03-15", got)
	}
}

func TestParseICalTime_Invalid(t *testing.T) {
	got := parseICalTime("not-a-date")
	if !got.IsZero() {
		t.Errorf("invalid input should return zero time, got %v", got)
	}
}

// --- ParseICal / FormatICal round-trip ---------------------------------------

const sampleICal = `BEGIN:VCALENDAR
VERSION:2.0
PRODID:-//Test//Test//EN
BEGIN:VEVENT
SUMMARY:Team meeting
DESCRIPTION:Weekly sync
LOCATION:Conference room A
DTSTART:20240315T100000Z
DTEND:20240315T110000Z
DTSTAMP:20240101T000000Z
UID:test-uid-001@test
END:VEVENT
END:VCALENDAR
`

func TestParseICal_BasicEvent(t *testing.T) {
	events, err := ParseICal(sampleICal)
	if err != nil {
		t.Fatalf("ParseICal: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("want 1 event, got %d", len(events))
	}
	ev := events[0]
	if ev.Title != "Team meeting" {
		t.Errorf("Title = %q, want %q", ev.Title, "Team meeting")
	}
	if ev.Description != "Weekly sync" {
		t.Errorf("Description = %q", ev.Description)
	}
	if ev.Location != "Conference room A" {
		t.Errorf("Location = %q", ev.Location)
	}
	if ev.AllDay {
		t.Error("should not be all-day")
	}
	wantStart := time.Date(2024, 3, 15, 10, 0, 0, 0, time.UTC)
	if !ev.StartsAt.Equal(wantStart) {
		t.Errorf("StartsAt = %v, want %v", ev.StartsAt, wantStart)
	}
}

func TestParseICal_AllDayEvent(t *testing.T) {
	src := `BEGIN:VCALENDAR
VERSION:2.0
BEGIN:VEVENT
SUMMARY:Holiday
DTSTART;VALUE=DATE:20240101
DTEND;VALUE=DATE:20240102
DTSTAMP:20240101T000000Z
UID:holiday-001@test
END:VEVENT
END:VCALENDAR
`
	events, err := ParseICal(src)
	if err != nil {
		t.Fatalf("ParseICal: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("want 1 event, got %d", len(events))
	}
	if !events[0].AllDay {
		t.Error("should be all-day event")
	}
	if events[0].Title != "Holiday" {
		t.Errorf("Title = %q", events[0].Title)
	}
	// end is exclusive in iCal (Jan 2), so stored as Jan 1
	if events[0].EndsAt.Day() != 1 {
		t.Errorf("AllDay EndsAt should be Jan 1, got %v", events[0].EndsAt)
	}
}

func TestParseICal_SkipsNonEvent(t *testing.T) {
	src := `BEGIN:VCALENDAR
VERSION:2.0
BEGIN:VTODO
SUMMARY:Task
DTSTART:20240315T100000Z
END:VTODO
BEGIN:VEVENT
SUMMARY:Real event
DTSTART:20240315T100000Z
DTEND:20240315T110000Z
DTSTAMP:20240101T000000Z
UID:real-001@test
END:VEVENT
END:VCALENDAR
`
	events, err := ParseICal(src)
	if err != nil {
		t.Fatalf("ParseICal: %v", err)
	}
	if len(events) != 1 {
		t.Errorf("want 1 event (VTODO skipped), got %d", len(events))
	}
}

func TestParseICal_MissingDTSTART(t *testing.T) {
	src := `BEGIN:VCALENDAR
VERSION:2.0
BEGIN:VEVENT
SUMMARY:No start
DTSTAMP:20240101T000000Z
UID:bad-001@test
END:VEVENT
END:VCALENDAR
`
	events, err := ParseICal(src)
	if err != nil {
		t.Fatalf("ParseICal: %v", err)
	}
	// Malformed event should be skipped.
	if len(events) != 0 {
		t.Errorf("want 0 events for missing DTSTART, got %d", len(events))
	}
}

func TestParseICal_InvalidInput(t *testing.T) {
	_, err := ParseICal("not ical at all")
	if err == nil {
		t.Error("expected error for invalid iCal input")
	}
}

func TestFormatICal_RoundTrip(t *testing.T) {
	original := []Event{
		{
			ID:          42,
			Title:       "Sprint review",
			Description: "Q1 review",
			Location:    "Zoom",
			StartsAt:    time.Date(2024, 6, 1, 14, 0, 0, 0, time.UTC),
			EndsAt:      time.Date(2024, 6, 1, 15, 0, 0, 0, time.UTC),
			AllDay:      false,
		},
	}

	ics := FormatICal(original)
	parsed, err := ParseICal(ics)
	if err != nil {
		t.Fatalf("ParseICal after FormatICal: %v", err)
	}
	if len(parsed) != 1 {
		t.Fatalf("want 1 event, got %d", len(parsed))
	}
	got := parsed[0]
	if got.Title != original[0].Title {
		t.Errorf("Title = %q, want %q", got.Title, original[0].Title)
	}
	if got.Description != original[0].Description {
		t.Errorf("Description = %q", got.Description)
	}
	if got.Location != original[0].Location {
		t.Errorf("Location = %q", got.Location)
	}
	if !got.StartsAt.Equal(original[0].StartsAt) {
		t.Errorf("StartsAt = %v, want %v", got.StartsAt, original[0].StartsAt)
	}
	if !got.EndsAt.Equal(original[0].EndsAt) {
		t.Errorf("EndsAt = %v, want %v", got.EndsAt, original[0].EndsAt)
	}
}

func TestFormatICal_AllDayRoundTrip(t *testing.T) {
	original := []Event{
		{
			ID:       1,
			Title:    "Company holiday",
			StartsAt: time.Date(2024, 12, 25, 0, 0, 0, 0, time.UTC),
			EndsAt:   time.Date(2024, 12, 25, 0, 0, 0, 0, time.UTC),
			AllDay:   true,
		},
	}
	ics := FormatICal(original)
	parsed, err := ParseICal(ics)
	if err != nil {
		t.Fatalf("ParseICal: %v", err)
	}
	if len(parsed) != 1 {
		t.Fatalf("want 1 event, got %d", len(parsed))
	}
	if !parsed[0].AllDay {
		t.Error("should be all-day after round-trip")
	}
	if parsed[0].StartsAt.Day() != 25 || parsed[0].StartsAt.Month() != 12 {
		t.Errorf("StartsAt = %v, want Dec 25", parsed[0].StartsAt)
	}
}

func TestFormatICal_ContainsRequiredFields(t *testing.T) {
	ics := FormatICal([]Event{{
		ID:       1,
		Title:    "Test",
		StartsAt: time.Date(2024, 1, 1, 9, 0, 0, 0, time.UTC),
		EndsAt:   time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC),
	}})
	for _, want := range []string{"BEGIN:VCALENDAR", "END:VCALENDAR", "BEGIN:VEVENT", "END:VEVENT", "SUMMARY:Test"} {
		if !strings.Contains(ics, want) {
			t.Errorf("iCal output missing %q", want)
		}
	}
}


func TestFormatICalSingle(t *testing.T) {
	ev := Event{ID: 1, Title: "Solo", StartsAt: time.Now(), EndsAt: time.Now().Add(time.Hour)}
	single := FormatICalSingle(ev)
	all := FormatICal([]Event{ev})
	if single != all {
		t.Error("FormatICalSingle should produce identical output to FormatICal with one event")
	}
}

// --- FormatICalInvite ---------------------------------------------------------

func sampleEvent() Event {
	return Event{
		ID:       7,
		Title:    "Team sync",
		Location: "Zoom",
		StartsAt: time.Date(2026, 5, 20, 14, 0, 0, 0, time.UTC),
		EndsAt:   time.Date(2026, 5, 20, 15, 0, 0, 0, time.UTC),
	}
}

func TestFormatICalInvite_MethodRequest(t *testing.T) {
	ics := FormatICalInvite(sampleEvent(), "alice@example.com", []string{"bob@example.com"})
	if !strings.Contains(ics, "METHOD:REQUEST") {
		t.Error("invite must contain METHOD:REQUEST")
	}
}

func TestFormatICalInvite_Organizer(t *testing.T) {
	ics := FormatICalInvite(sampleEvent(), "alice@example.com", nil)
	if !strings.Contains(ics, "mailto:alice@example.com") {
		t.Error("invite must contain ORGANIZER with mailto: prefix")
	}
	if !strings.Contains(ics, "ORGANIZER") {
		t.Error("invite must contain ORGANIZER property")
	}
}

func TestFormatICalInvite_Attendees(t *testing.T) {
	ics := FormatICalInvite(sampleEvent(), "alice@example.com", []string{"bob@example.com", "carol@example.com"})
	if !strings.Contains(ics, "mailto:bob@example.com") {
		t.Error("missing bob attendee")
	}
	if !strings.Contains(ics, "mailto:carol@example.com") {
		t.Error("missing carol attendee")
	}
	if !strings.Contains(ics, "RSVP=TRUE") {
		t.Error("attendees should have RSVP=TRUE")
	}
	if !strings.Contains(ics, "PARTSTAT=NEEDS-ACTION") {
		t.Error("attendees should have PARTSTAT=NEEDS-ACTION")
	}
}

func TestFormatICalInvite_OrganizerNotInAttendees(t *testing.T) {
	// The organizer's own address should not appear as an ATTENDEE.
	ics := FormatICalInvite(sampleEvent(), "alice@example.com", []string{"alice@example.com", "bob@example.com"})
	// Count occurrences of alice's address in ATTENDEE lines only.
	lines := strings.Split(ics, "\n")
	attendeeAlice := 0
	for _, l := range lines {
		if strings.HasPrefix(l, "ATTENDEE") && strings.Contains(l, "alice@example.com") {
			attendeeAlice++
		}
	}
	if attendeeAlice > 0 {
		t.Error("organizer address should not appear as ATTENDEE")
	}
}

func TestFormatICalInvite_DeduplicatesAttendees(t *testing.T) {
	ics := FormatICalInvite(sampleEvent(), "alice@example.com", []string{"bob@example.com", "BOB@EXAMPLE.COM"})
	count := strings.Count(ics, "mailto:bob@example.com")
	if count != 1 {
		t.Errorf("duplicate attendees should be deduplicated, got %d occurrences", count)
	}
}

func TestFormatICalInvite_NoAttendees(t *testing.T) {
	ics := FormatICalInvite(sampleEvent(), "alice@example.com", nil)
	if !strings.Contains(ics, "METHOD:REQUEST") {
		t.Error("invite without attendees must still have METHOD:REQUEST")
	}
	if strings.Contains(ics, "ATTENDEE") {
		t.Error("no ATTENDEE lines expected when attendees list is empty")
	}
}

func TestFormatICalInvite_EventFields(t *testing.T) {
	ev := sampleEvent()
	ics := FormatICalInvite(ev, "alice@example.com", []string{"bob@example.com"})
	if !strings.Contains(ics, "SUMMARY:Team sync") {
		t.Error("invite must contain the event summary")
	}
	if !strings.Contains(ics, "LOCATION:Zoom") {
		t.Error("invite must contain the event location")
	}
	if !strings.Contains(ics, "STATUS:CONFIRMED") {
		t.Error("invite should have STATUS:CONFIRMED")
	}
}
