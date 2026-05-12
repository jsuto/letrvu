// Package calendar manages calendar events.
package calendar

import (
	"database/sql"
	"fmt"
	"sort"
	"time"

	"github.com/jsuto/letrvu/internal/db"
)

// Event represents a calendar event.
type Event struct {
	ID          int64     `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Location    string    `json:"location"`
	AllDay      bool      `json:"all_day"`
	StartsAt    time.Time `json:"starts_at"`
	EndsAt      time.Time `json:"ends_at"`
	Rrule       string    `json:"rrule,omitempty"`
}

// Store is the calendar data layer.
type Store struct {
	db *db.DB
}

// NewStore creates a Store backed by the given database.
func NewStore(database *db.DB) *Store {
	return &Store{db: database}
}

// List returns events that overlap [from, to], expanding recurring events into
// individual occurrences within that window.
func (s *Store) List(owner, imapHost string, from, to time.Time) ([]Event, error) {
	toStr := to.UTC().Format(time.RFC3339)
	fromStr := from.UTC().Format(time.RFC3339)

	// Non-recurring events that overlap [from, to].
	rows1, err := s.db.Query(
		s.db.Q(`SELECT id, title, description, location, all_day, starts_at, ends_at, rrule
			FROM calendar_events
			WHERE owner=? AND imap_host=? AND rrule=''
			  AND starts_at <= ? AND ends_at >= ?
			ORDER BY starts_at`),
		owner, imapHost, toStr, fromStr,
	)
	if err != nil {
		return nil, err
	}
	result := []Event{}
	for rows1.Next() {
		e, err := scanEvent(rows1)
		if err != nil {
			rows1.Close()
			return nil, err
		}
		result = append(result, e)
	}
	rows1.Close()
	if err := rows1.Err(); err != nil {
		return nil, err
	}

	// Recurring events whose DTSTART is at or before the window end.
	rows2, err := s.db.Query(
		s.db.Q(`SELECT id, title, description, location, all_day, starts_at, ends_at, rrule
			FROM calendar_events
			WHERE owner=? AND imap_host=? AND rrule!=''
			  AND starts_at <= ?
			ORDER BY starts_at`),
		owner, imapHost, toStr,
	)
	if err != nil {
		return nil, err
	}
	var recurring []Event
	for rows2.Next() {
		e, err := scanEvent(rows2)
		if err != nil {
			rows2.Close()
			return nil, err
		}
		recurring = append(recurring, e)
	}
	rows2.Close()
	if err := rows2.Err(); err != nil {
		return nil, err
	}

	// Expand each recurring event into occurrences within [from, to].
	for _, base := range recurring {
		duration := base.EndsAt.Sub(base.StartsAt)
		starts, err := expandRRule(base.Rrule, base.StartsAt, duration, from, to)
		if err != nil {
			continue // skip events with malformed RRULE
		}
		for _, occ := range starts {
			ev := base
			ev.StartsAt = occ
			ev.EndsAt = occ.Add(duration)
			result = append(result, ev)
		}
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].StartsAt.Before(result[j].StartsAt)
	})
	return result, nil
}

// ListAll returns all base events for the owner without expanding recurrences.
// Used for export so the RRULE string is preserved in the output.
func (s *Store) ListAll(owner, imapHost string) ([]Event, error) {
	rows, err := s.db.Query(
		s.db.Q(`SELECT id, title, description, location, all_day, starts_at, ends_at, rrule
			FROM calendar_events
			WHERE owner=? AND imap_host=?
			ORDER BY starts_at`),
		owner, imapHost,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	events := []Event{}
	for rows.Next() {
		e, err := scanEvent(rows)
		if err != nil {
			return nil, err
		}
		events = append(events, e)
	}
	return events, rows.Err()
}

// Get returns a single event by ID, verifying ownership.
func (s *Store) Get(id int64, owner, imapHost string) (*Event, error) {
	row := s.db.QueryRow(
		s.db.Q(`SELECT id, title, description, location, all_day, starts_at, ends_at, rrule
			FROM calendar_events WHERE id=? AND owner=? AND imap_host=?`),
		id, owner, imapHost,
	)
	e, err := scanEvent(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &e, nil
}

// Create inserts a new event and returns it.
func (s *Store) Create(owner, imapHost string, ev Event) (*Event, error) {
	id, err := s.db.InsertReturningID(
		`INSERT INTO calendar_events (owner, imap_host, title, description, location, all_day, starts_at, ends_at, rrule)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		owner, imapHost,
		ev.Title, ev.Description, ev.Location,
		boolToInt(ev.AllDay),
		ev.StartsAt.UTC().Format(time.RFC3339),
		ev.EndsAt.UTC().Format(time.RFC3339),
		ev.Rrule,
	)
	if err != nil {
		return nil, fmt.Errorf("insert event: %w", err)
	}
	return s.Get(id, owner, imapHost)
}

// Update replaces an event's fields.
func (s *Store) Update(id int64, owner, imapHost string, ev Event) (*Event, error) {
	res, err := s.db.Exec(
		s.db.Q(`UPDATE calendar_events
			SET title=?, description=?, location=?, all_day=?, starts_at=?, ends_at=?, rrule=?
			WHERE id=? AND owner=? AND imap_host=?`),
		ev.Title, ev.Description, ev.Location,
		boolToInt(ev.AllDay),
		ev.StartsAt.UTC().Format(time.RFC3339),
		ev.EndsAt.UTC().Format(time.RFC3339),
		ev.Rrule,
		id, owner, imapHost,
	)
	if err != nil {
		return nil, err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return nil, nil
	}
	return s.Get(id, owner, imapHost)
}

// Delete removes an event.
func (s *Store) Delete(id int64, owner, imapHost string) error {
	_, err := s.db.Exec(
		s.db.Q(`DELETE FROM calendar_events WHERE id=? AND owner=? AND imap_host=?`),
		id, owner, imapHost,
	)
	return err
}

// scanner is satisfied by both *sql.Row and *sql.Rows.
type scanner interface {
	Scan(dest ...any) error
}

func scanEvent(s scanner) (Event, error) {
	var e Event
	var startsAt, endsAt string
	var allDay int
	err := s.Scan(&e.ID, &e.Title, &e.Description, &e.Location, &allDay, &startsAt, &endsAt, &e.Rrule)
	if err != nil {
		return Event{}, err
	}
	e.AllDay = allDay != 0
	e.StartsAt, _ = time.Parse(time.RFC3339, startsAt)
	e.EndsAt, _ = time.Parse(time.RFC3339, endsAt)
	return e, nil
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
