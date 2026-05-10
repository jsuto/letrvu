// Package contacts manages the address book.
package contacts

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/yourusername/letrvu/internal/db"
)

// Contact represents an address book entry.
type Contact struct {
	ID     int64          `json:"id"`
	Name   string         `json:"name"`
	Notes  string         `json:"notes"`
	Emails []ContactEmail `json:"emails"`
}

// ContactEmail is a single email address belonging to a contact.
type ContactEmail struct {
	ID    int64  `json:"id"`
	Email string `json:"email"`
	Label string `json:"label"`
}

// AutocompleteResult is a lightweight hit for the compose address field.
type AutocompleteResult struct {
	ContactID int64  `json:"contact_id"`
	Name      string `json:"name"`
	Email     string `json:"email"`
}

// Store is the address book data layer.
type Store struct {
	db *db.DB
}

// NewStore creates a Store backed by the given database.
func NewStore(database *db.DB) *Store {
	return &Store{db: database}
}

// List returns all contacts (with their emails) for the given owner.
func (s *Store) List(owner, imapHost string) ([]Contact, error) {
	rows, err := s.db.Query(
		s.db.Q(`SELECT id, name, notes FROM contacts WHERE owner=? AND imap_host=? ORDER BY name`),
		owner, imapHost,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var contacts []Contact
	for rows.Next() {
		var c Contact
		if err := rows.Scan(&c.ID, &c.Name, &c.Notes); err != nil {
			return nil, err
		}
		contacts = append(contacts, c)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	for i := range contacts {
		emails, err := s.listEmails(contacts[i].ID)
		if err != nil {
			return nil, err
		}
		contacts[i].Emails = emails
	}
	return contacts, nil
}

// Get returns a single contact by ID, verifying ownership.
func (s *Store) Get(id int64, owner, imapHost string) (*Contact, error) {
	var c Contact
	err := s.db.QueryRow(
		s.db.Q(`SELECT id, name, notes FROM contacts WHERE id=? AND owner=? AND imap_host=?`),
		id, owner, imapHost,
	).Scan(&c.ID, &c.Name, &c.Notes)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	emails, err := s.listEmails(c.ID)
	if err != nil {
		return nil, err
	}
	c.Emails = emails
	return &c, nil
}

// Create inserts a new contact and returns its ID.
func (s *Store) Create(owner, imapHost, name, notes string, emails []ContactEmail) (*Contact, error) {
	id, err := s.db.InsertReturningID(
		`INSERT INTO contacts (owner, imap_host, name, notes) VALUES (?, ?, ?, ?)`,
		owner, imapHost, name, notes,
	)
	if err != nil {
		return nil, fmt.Errorf("insert contact: %w", err)
	}
	for _, e := range emails {
		if _, err := s.db.InsertReturningID(
			`INSERT INTO contact_emails (contact_id, email, label) VALUES (?, ?, ?)`,
			id, strings.ToLower(strings.TrimSpace(e.Email)), e.Label,
		); err != nil {
			return nil, fmt.Errorf("insert email: %w", err)
		}
	}
	return s.Get(id, owner, imapHost)
}

// Update replaces name, notes, and email list for a contact.
func (s *Store) Update(id int64, owner, imapHost, name, notes string, emails []ContactEmail) (*Contact, error) {
	res, err := s.db.Exec(
		s.db.Q(`UPDATE contacts SET name=?, notes=? WHERE id=? AND owner=? AND imap_host=?`),
		name, notes, id, owner, imapHost,
	)
	if err != nil {
		return nil, err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return nil, nil // not found or not owned
	}

	// Replace email list.
	if _, err := s.db.Exec(s.db.Q(`DELETE FROM contact_emails WHERE contact_id=?`), id); err != nil {
		return nil, err
	}
	for _, e := range emails {
		if _, err := s.db.InsertReturningID(
			`INSERT INTO contact_emails (contact_id, email, label) VALUES (?, ?, ?)`,
			id, strings.ToLower(strings.TrimSpace(e.Email)), e.Label,
		); err != nil {
			return nil, fmt.Errorf("insert email: %w", err)
		}
	}
	return s.Get(id, owner, imapHost)
}

// Delete removes a contact and its emails.
func (s *Store) Delete(id int64, owner, imapHost string) error {
	// Verify ownership first.
	var dummy int64
	err := s.db.QueryRow(
		s.db.Q(`SELECT id FROM contacts WHERE id=? AND owner=? AND imap_host=?`),
		id, owner, imapHost,
	).Scan(&dummy)
	if err == sql.ErrNoRows {
		return nil
	}
	if err != nil {
		return err
	}
	if _, err := s.db.Exec(s.db.Q(`DELETE FROM contact_emails WHERE contact_id=?`), id); err != nil {
		return err
	}
	_, err = s.db.Exec(s.db.Q(`DELETE FROM contacts WHERE id=?`), id)
	return err
}

// Autocomplete returns up to 10 contacts whose name or email starts with prefix.
func (s *Store) Autocomplete(owner, imapHost, prefix string) ([]AutocompleteResult, error) {
	like := strings.ToLower(prefix) + "%"
	rows, err := s.db.Query(
		s.db.Q(`SELECT c.id, c.name, ce.email
			FROM contacts c
			JOIN contact_emails ce ON ce.contact_id = c.id
			WHERE c.owner=? AND c.imap_host=?
			  AND (LOWER(c.name) LIKE ? OR LOWER(ce.email) LIKE ?)
			ORDER BY c.name, ce.email
			LIMIT 10`),
		owner, imapHost, like, like,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []AutocompleteResult
	for rows.Next() {
		var r AutocompleteResult
		if err := rows.Scan(&r.ContactID, &r.Name, &r.Email); err != nil {
			return nil, err
		}
		results = append(results, r)
	}
	return results, rows.Err()
}

// FindByEmail looks up a contact by exact email address.
func (s *Store) FindByEmail(owner, imapHost, email string) (*Contact, error) {
	var contactID int64
	err := s.db.QueryRow(
		s.db.Q(`SELECT c.id FROM contacts c
			JOIN contact_emails ce ON ce.contact_id = c.id
			WHERE c.owner=? AND c.imap_host=? AND LOWER(ce.email)=?`),
		owner, imapHost, strings.ToLower(email),
	).Scan(&contactID)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return s.Get(contactID, owner, imapHost)
}

// SaveFromMessage creates a new contact or adds the email to an existing one.
// It is idempotent — calling it twice with the same email is safe.
func (s *Store) SaveFromMessage(owner, imapHost, name, email string) (*Contact, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	existing, err := s.FindByEmail(owner, imapHost, email)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return existing, nil // already in address book
	}
	return s.Create(owner, imapHost, name, "", []ContactEmail{{Email: email}})
}

func (s *Store) listEmails(contactID int64) ([]ContactEmail, error) {
	rows, err := s.db.Query(
		s.db.Q(`SELECT id, email, label FROM contact_emails WHERE contact_id=? ORDER BY id`),
		contactID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	emails := []ContactEmail{}
	for rows.Next() {
		var e ContactEmail
		if err := rows.Scan(&e.ID, &e.Email, &e.Label); err != nil {
			return nil, err
		}
		emails = append(emails, e)
	}
	return emails, rows.Err()
}
