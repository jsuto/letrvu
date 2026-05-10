// Package settings persists per-user configuration in the database.
package settings

import (
	"fmt"

	"github.com/jsuto/letrvu/internal/db"
)

// allowed is the set of setting keys users may read and write.
var allowed = map[string]bool{
	"display_name": true,
	"signature":    true,
	"identities":   true,
}

// Store persists per-user settings keyed by (username, imap_host).
type Store struct {
	db *db.DB
}

func NewStore(database *db.DB) *Store {
	return &Store{db: database}
}

// Get returns all settings for the given user and IMAP host.
func (s *Store) Get(username, imapHost string) (map[string]string, error) {
	rows, err := s.db.Query(
		s.db.Q(`SELECT key, value FROM user_settings WHERE username = ? AND imap_host = ?`),
		username, imapHost,
	)
	if err != nil {
		return nil, fmt.Errorf("get settings: %w", err)
	}
	defer rows.Close()

	result := make(map[string]string)
	for rows.Next() {
		var k, v string
		if err := rows.Scan(&k, &v); err != nil {
			return nil, err
		}
		result[k] = v
	}
	return result, rows.Err()
}

// Set upserts one or more settings. Unknown keys are silently ignored.
func (s *Store) Set(username, imapHost string, values map[string]string) error {
	for k, v := range values {
		if !allowed[k] {
			continue
		}
		var err error
		if s.db.Driver == "postgres" {
			_, err = s.db.Exec(
				`INSERT INTO user_settings (username, imap_host, key, value)
				 VALUES ($1, $2, $3, $4)
				 ON CONFLICT (username, imap_host, key) DO UPDATE SET value = EXCLUDED.value`,
				username, imapHost, k, v,
			)
		} else {
			_, err = s.db.Exec(
				`INSERT OR REPLACE INTO user_settings (username, imap_host, key, value) VALUES (?, ?, ?, ?)`,
				username, imapHost, k, v,
			)
		}
		if err != nil {
			return fmt.Errorf("set %q: %w", k, err)
		}
	}
	return nil
}
