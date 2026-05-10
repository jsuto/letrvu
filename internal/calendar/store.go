// Package calendar manages calendar events.
package calendar

import (
	"database/sql"
	"fmt"
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
}

// Store is the calendar data layer.
type Store struct {
	db *db.DB
}

// NewStore creates a Store backed by the given database.
func NewStore(database *db.DB) *Store {
	return &Store{db: database}
}

// List returns events for the given owner that overlap the [from, to] range.
func (s *Store) List(owner, imapHost string, from, to time.Time) ([]Event, error) {
	rows, err := s.db.Query(
		s.db.Q(`SELECT id, title, description, location, all_day, starts_at, ends_at
			FROM calendar_events
			WHERE owner=? AND imap_host=?
			  AND starts_at <= ? AND ends_at >= ?
			ORDER BY starts_at`),
		owner, imapHost,
		to.UTC().Format(time.RFC3339),
		from.UTC().Format(time.RFC3339),
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
		s.db.Q(`SELECT id, title, description, location, all_day, starts_at, ends_at
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
		`INSERT INTO calendar_events (owner, imap_host, title, description, location, all_day, starts_at, ends_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		owner, imapHost,
		ev.Title, ev.Description, ev.Location,
		boolToInt(ev.AllDay),
		ev.StartsAt.UTC().Format(time.RFC3339),
		ev.EndsAt.UTC().Format(time.RFC3339),
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
			SET title=?, description=?, location=?, all_day=?, starts_at=?, ends_at=?
			WHERE id=? AND owner=? AND imap_host=?`),
		ev.Title, ev.Description, ev.Location,
		boolToInt(ev.AllDay),
		ev.StartsAt.UTC().Format(time.RFC3339),
		ev.EndsAt.UTC().Format(time.RFC3339),
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
	err := s.Scan(&e.ID, &e.Title, &e.Description, &e.Location, &allDay, &startsAt, &endsAt)
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
