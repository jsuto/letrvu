// Package index maintains a local database cache of IMAP message headers so
// that cross-folder search can be satisfied without touching the IMAP server.
package index

import (
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jsuto/letrvu/internal/db"
	"github.com/jsuto/letrvu/internal/imap"
)

// Store is the message index data layer.
type Store struct {
	db *db.DB
}

// NewStore creates a Store backed by the given database.
func NewStore(database *db.DB) *Store {
	return &Store{db: database}
}

// Upsert inserts or replaces message summaries for the given folder.
func (s *Store) Upsert(username, imapHost, folder string, msgs []imap.Message) error {
	for _, m := range msgs {
		var err error
		if s.db.Driver == "postgres" {
			_, err = s.db.Exec(
				`INSERT INTO message_index
					(username, imap_host, folder, uid, subject, from_addr, date,
					 read, flagged, has_attachments, size, message_id, in_reply_to, refs)
				 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)
				 ON CONFLICT (username, imap_host, folder, uid)
				 DO UPDATE SET
					subject=EXCLUDED.subject, from_addr=EXCLUDED.from_addr, date=EXCLUDED.date,
					read=EXCLUDED.read, flagged=EXCLUDED.flagged,
					has_attachments=EXCLUDED.has_attachments, size=EXCLUDED.size,
					message_id=EXCLUDED.message_id, in_reply_to=EXCLUDED.in_reply_to,
					refs=EXCLUDED.refs`,
				username, imapHost, folder, m.UID,
				safeUTF8(m.Subject), safeUTF8(m.From), m.Date.UTC().Format(time.RFC3339),
				boolInt(m.Read), boolInt(m.Flagged), boolInt(m.HasAttachments), m.Size,
				m.MessageID, m.InReplyTo, m.References,
			)
		} else {
			_, err = s.db.Exec(
				`INSERT OR REPLACE INTO message_index
					(username, imap_host, folder, uid, subject, from_addr, date,
					 read, flagged, has_attachments, size, message_id, in_reply_to, refs)
				 VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
				username, imapHost, folder, m.UID,
				safeUTF8(m.Subject), safeUTF8(m.From), m.Date.UTC().Format(time.RFC3339),
				boolInt(m.Read), boolInt(m.Flagged), boolInt(m.HasAttachments), m.Size,
				m.MessageID, m.InReplyTo, m.References,
			)
		}
		if err != nil {
			return fmt.Errorf("upsert uid %d in %q: %w", m.UID, folder, err)
		}
	}
	return nil
}

// UpdateRead updates the read flag for a single message.
func (s *Store) UpdateRead(username, imapHost, folder string, uid uint32, read bool) {
	_, _ = s.db.Exec(
		s.db.Q(`UPDATE message_index SET read=? WHERE username=? AND imap_host=? AND folder=? AND uid=?`),
		boolInt(read), username, imapHost, folder, uid,
	)
}

// UpdateFlagged updates the flagged flag for a single message.
func (s *Store) UpdateFlagged(username, imapHost, folder string, uid uint32, flagged bool) {
	_, _ = s.db.Exec(
		s.db.Q(`UPDATE message_index SET flagged=? WHERE username=? AND imap_host=? AND folder=? AND uid=?`),
		boolInt(flagged), username, imapHost, folder, uid,
	)
}

// Delete removes specific UIDs from the index for a folder.
func (s *Store) Delete(username, imapHost, folder string, uids []uint32) error {
	if len(uids) == 0 {
		return nil
	}
	placeholders := strings.Repeat(",?", len(uids))[1:]
	args := []any{username, imapHost, folder}
	for _, u := range uids {
		args = append(args, u)
	}
	_, err := s.db.Exec(
		s.db.Q(fmt.Sprintf(
			`DELETE FROM message_index WHERE username=? AND imap_host=? AND folder=? AND uid IN (%s)`,
			placeholders,
		)),
		args...,
	)
	return err
}

// DeleteFolder removes all index entries for a folder.
func (s *Store) DeleteFolder(username, imapHost, folder string) error {
	_, err := s.db.Exec(
		s.db.Q(`DELETE FROM message_index WHERE username=? AND imap_host=? AND folder=?`),
		username, imapHost, folder,
	)
	return err
}

// KnownUIDs returns the set of UIDs currently in the index for a folder.
func (s *Store) KnownUIDs(username, imapHost, folder string) (map[uint32]struct{}, error) {
	rows, err := s.db.Query(
		s.db.Q(`SELECT uid FROM message_index WHERE username=? AND imap_host=? AND folder=?`),
		username, imapHost, folder,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make(map[uint32]struct{})
	for rows.Next() {
		var uid uint32
		if err := rows.Scan(&uid); err != nil {
			return nil, err
		}
		result[uid] = struct{}{}
	}
	return result, rows.Err()
}

// UIDValidity returns the stored UID validity for a folder (0 if unknown).
func (s *Store) UIDValidity(username, imapHost, folder string) uint32 {
	var v uint32
	_ = s.db.QueryRow(
		s.db.Q(`SELECT uid_validity FROM folder_index_state WHERE username=? AND imap_host=? AND folder=?`),
		username, imapHost, folder,
	).Scan(&v)
	return v
}

// SetUIDValidity persists the UID validity for a folder.
func (s *Store) SetUIDValidity(username, imapHost, folder string, v uint32) {
	if s.db.Driver == "postgres" {
		_, _ = s.db.Exec(
			`INSERT INTO folder_index_state (username, imap_host, folder, uid_validity)
			 VALUES ($1,$2,$3,$4)
			 ON CONFLICT (username, imap_host, folder) DO UPDATE SET uid_validity=EXCLUDED.uid_validity`,
			username, imapHost, folder, v,
		)
	} else {
		_, _ = s.db.Exec(
			`INSERT OR REPLACE INTO folder_index_state (username, imap_host, folder, uid_validity) VALUES (?,?,?,?)`,
			username, imapHost, folder, v,
		)
	}
}

// Search returns messages whose subject or sender matches query, newest first.
func (s *Store) Search(username, imapHost, query string) ([]imap.Message, error) {
	pattern := "%" + query + "%"
	var rows interface {
		Next() bool
		Scan(...any) error
		Close() error
		Err() error
	}
	var err error

	if s.db.Driver == "postgres" {
		rows, err = s.db.Query(
			`SELECT folder, uid, subject, from_addr, date, read, flagged, has_attachments,
				size, message_id, in_reply_to, refs
			 FROM message_index
			 WHERE username=$1 AND imap_host=$2 AND (subject ILIKE $3 OR from_addr ILIKE $3)
			 ORDER BY date DESC
			 LIMIT 200`,
			username, imapHost, pattern,
		)
	} else {
		rows, err = s.db.Query(
			`SELECT folder, uid, subject, from_addr, date, read, flagged, has_attachments,
				size, message_id, in_reply_to, refs
			 FROM message_index
			 WHERE username=? AND imap_host=? AND (subject LIKE ? OR from_addr LIKE ?)
			 ORDER BY date DESC
			 LIMIT 200`,
			username, imapHost, pattern, pattern,
		)
	}
	if err != nil {
		return nil, fmt.Errorf("search index: %w", err)
	}
	defer rows.Close()

	var result []imap.Message
	for rows.Next() {
		var m imap.Message
		var dateStr string
		var readInt, flaggedInt, attachInt int
		if err := rows.Scan(
			&m.Folder, &m.UID, &m.Subject, &m.From, &dateStr,
			&readInt, &flaggedInt, &attachInt, &m.Size,
			&m.MessageID, &m.InReplyTo, &m.References,
		); err != nil {
			return nil, err
		}
		m.Date, _ = time.Parse(time.RFC3339, dateStr)
		m.Read = readInt != 0
		m.Flagged = flaggedInt != 0
		m.HasAttachments = attachInt != 0
		result = append(result, m)
	}
	return result, rows.Err()
}

// IndexedCount returns how many messages are indexed for the user.
// Used to detect an empty index (first run).
func (s *Store) IndexedCount(username, imapHost string) int {
	var n int
	_ = s.db.QueryRow(
		s.db.Q(`SELECT COUNT(*) FROM message_index WHERE username=? AND imap_host=?`),
		username, imapHost,
	).Scan(&n)
	return n
}

func boolInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

// safeUTF8 strips bytes that form invalid UTF-8 sequences.
// PostgreSQL requires valid UTF-8; some email headers contain legacy encodings.
func safeUTF8(s string) string {
	if utf8.ValidString(s) {
		return s
	}
	return strings.ToValidUTF8(s, "")
}
