// Package templates provides storage for per-user message templates (canned responses).
package templates

import (
	"errors"
	"fmt"

	"github.com/jsuto/letrvu/internal/db"
)

// Template is a saved message template belonging to a user.
type Template struct {
	ID      int64  `json:"id"`
	Name    string `json:"name"`
	Subject string `json:"subject"`
	Body    string `json:"body"` // HTML
}

// Store provides CRUD operations for message templates.
type Store struct {
	db *db.DB
}

// NewStore creates a Store backed by db.
func NewStore(db *db.DB) *Store {
	return &Store{db: db}
}

// List returns all templates for the given user, ordered by name.
func (s *Store) List(username, imapHost string) ([]Template, error) {
	rows, err := s.db.Query(
		s.db.Q(`SELECT id, name, subject, body
		         FROM message_templates
		         WHERE owner = ? AND imap_host = ?
		         ORDER BY name`),
		username, imapHost,
	)
	if err != nil {
		return nil, fmt.Errorf("templates list: %w", err)
	}
	defer rows.Close()

	var out []Template
	for rows.Next() {
		var t Template
		if err := rows.Scan(&t.ID, &t.Name, &t.Subject, &t.Body); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

// Create inserts a new template and returns its ID.
func (s *Store) Create(username, imapHost string, t Template) (int64, error) {
	return s.db.InsertReturningID(
		`INSERT INTO message_templates (owner, imap_host, name, subject, body)
		 VALUES (?, ?, ?, ?, ?)`,
		username, imapHost, t.Name, t.Subject, t.Body,
	)
}

// Update replaces a template's fields.
func (s *Store) Update(id int64, username, imapHost string, t Template) error {
	res, err := s.db.Exec(
		s.db.Q(`UPDATE message_templates
		         SET name = ?, subject = ?, body = ?
		         WHERE id = ? AND owner = ? AND imap_host = ?`),
		t.Name, t.Subject, t.Body, id, username, imapHost,
	)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("template %d not found", id)
	}
	return nil
}

// Delete removes a template by ID.
func (s *Store) Delete(id int64, username, imapHost string) error {
	res, err := s.db.Exec(
		s.db.Q(`DELETE FROM message_templates WHERE id = ? AND owner = ? AND imap_host = ?`),
		id, username, imapHost,
	)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return errors.New("template not found")
	}
	return nil
}
