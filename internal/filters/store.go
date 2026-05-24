// Package filters provides storage and evaluation for per-user mail filter rules.
package filters

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jsuto/letrvu/internal/db"
)

// Condition describes a single test applied to an incoming message.
type Condition struct {
	Field string `json:"field"` // subject | from | to | body | has_attachment | size
	Op    string `json:"op"`    // contains | not_contains | equals | not_equals | matches
	Value string `json:"value"`
}

// Action describes what to do when a filter matches.
type Action struct {
	Type  string `json:"type"`  // move | mark_read | mark_flagged | delete | stop
	Value string `json:"value"` // destination folder for "move"; unused for others
}

// Filter is a single mail filter rule belonging to a user.
type Filter struct {
	ID         int64       `json:"id"`
	Position   int         `json:"position"`
	Name       string      `json:"name"`
	MatchAll   bool        `json:"match_all"` // true = AND, false = OR
	Conditions []Condition `json:"conditions"`
	Actions    []Action    `json:"actions"`
	Enabled    bool        `json:"enabled"`
}

// Store provides CRUD operations for mail filters.
type Store struct {
	db *db.DB
}

// NewStore creates a Store backed by db.
func NewStore(db *db.DB) *Store {
	return &Store{db: db}
}

// List returns all filters for the given user, ordered by position.
func (s *Store) List(username, imapHost string) ([]Filter, error) {
	rows, err := s.db.Query(
		s.db.Q(`SELECT id, position, name, match_all, conditions, actions, enabled
		         FROM mail_filters
		         WHERE owner = ? AND imap_host = ?
		         ORDER BY position, id`),
		username, imapHost,
	)
	if err != nil {
		return nil, fmt.Errorf("filters list: %w", err)
	}
	defer rows.Close()

	var out []Filter
	for rows.Next() {
		var f Filter
		var condJSON, actJSON string
		var matchAll, enabled int
		if err := rows.Scan(&f.ID, &f.Position, &f.Name, &matchAll, &condJSON, &actJSON, &enabled); err != nil {
			return nil, err
		}
		f.MatchAll = matchAll != 0
		f.Enabled = enabled != 0
		if err := json.Unmarshal([]byte(condJSON), &f.Conditions); err != nil {
			f.Conditions = nil
		}
		if err := json.Unmarshal([]byte(actJSON), &f.Actions); err != nil {
			f.Actions = nil
		}
		out = append(out, f)
	}
	return out, rows.Err()
}

// Get returns a single filter by ID.
func (s *Store) Get(id int64, username, imapHost string) (Filter, error) {
	var f Filter
	var condJSON, actJSON string
	var matchAll, enabled int
	err := s.db.QueryRow(
		s.db.Q(`SELECT id, position, name, match_all, conditions, actions, enabled
		         FROM mail_filters WHERE id = ? AND owner = ? AND imap_host = ?`),
		id, username, imapHost,
	).Scan(&f.ID, &f.Position, &f.Name, &matchAll, &condJSON, &actJSON, &enabled)
	if errors.Is(err, sql.ErrNoRows) {
		return f, fmt.Errorf("filter %d not found", id)
	}
	if err != nil {
		return f, err
	}
	f.MatchAll = matchAll != 0
	f.Enabled = enabled != 0
	json.Unmarshal([]byte(condJSON), &f.Conditions) //nolint:errcheck
	json.Unmarshal([]byte(actJSON), &f.Actions)     //nolint:errcheck
	return f, nil
}

// Create inserts a new filter and returns its ID.
func (s *Store) Create(username, imapHost string, f Filter) (int64, error) {
	condJSON, err := json.Marshal(f.Conditions)
	if err != nil {
		return 0, err
	}
	actJSON, err := json.Marshal(f.Actions)
	if err != nil {
		return 0, err
	}
	matchAll := 0
	if f.MatchAll {
		matchAll = 1
	}
	enabled := 1
	if !f.Enabled {
		enabled = 0
	}

	// Place at end: max(position)+1
	var maxPos sql.NullInt64
	s.db.QueryRow( //nolint:errcheck
		s.db.Q(`SELECT MAX(position) FROM mail_filters WHERE owner = ? AND imap_host = ?`),
		username, imapHost,
	).Scan(&maxPos)
	pos := int(maxPos.Int64) + 1

	id, err := s.db.InsertReturningID(
		`INSERT INTO mail_filters (owner, imap_host, position, name, match_all, conditions, actions, enabled)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		username, imapHost, pos, f.Name, matchAll, string(condJSON), string(actJSON), enabled,
	)
	return id, err
}

// Update replaces a filter's mutable fields.
func (s *Store) Update(id int64, username, imapHost string, f Filter) error {
	condJSON, err := json.Marshal(f.Conditions)
	if err != nil {
		return err
	}
	actJSON, err := json.Marshal(f.Actions)
	if err != nil {
		return err
	}
	matchAll := 0
	if f.MatchAll {
		matchAll = 1
	}
	enabled := 1
	if !f.Enabled {
		enabled = 0
	}
	res, err := s.db.Exec(
		s.db.Q(`UPDATE mail_filters
		         SET name = ?, match_all = ?, conditions = ?, actions = ?, enabled = ?
		         WHERE id = ? AND owner = ? AND imap_host = ?`),
		f.Name, matchAll, string(condJSON), string(actJSON), enabled,
		id, username, imapHost,
	)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("filter %d not found", id)
	}
	return nil
}

// Delete removes a filter by ID.
func (s *Store) Delete(id int64, username, imapHost string) error {
	_, err := s.db.Exec(
		s.db.Q(`DELETE FROM mail_filters WHERE id = ? AND owner = ? AND imap_host = ?`),
		id, username, imapHost,
	)
	return err
}

// Reorder updates the position of each filter in the given ordered list of IDs.
func (s *Store) Reorder(ids []int64, username, imapHost string) error {
	for i, id := range ids {
		if _, err := s.db.Exec(
			s.db.Q(`UPDATE mail_filters SET position = ? WHERE id = ? AND owner = ? AND imap_host = ?`),
			i, id, username, imapHost,
		); err != nil {
			return err
		}
	}
	return nil
}
