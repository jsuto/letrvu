// Package db provides a thin database/sql wrapper with cross-driver
// placeholder translation and schema migration.
package db

import (
	"database/sql"
	"fmt"
	"strconv"
	"strings"

	_ "github.com/jackc/pgx/v5/stdlib" // registers "pgx"
	_ "modernc.org/sqlite"              // registers "sqlite"
)

// DB wraps *sql.DB and carries the driver name for query adaptation.
type DB struct {
	*sql.DB
	Driver string
}

// Open opens and pings a database connection.
// driver must be "postgres" or "sqlite".
func Open(driver, dsn string) (*DB, error) {
	sqlDriver := driver
	if driver == "postgres" {
		sqlDriver = "pgx"
	}
	db, err := sql.Open(sqlDriver, dsn)
	if err != nil {
		return nil, fmt.Errorf("open db (%s): %w", driver, err)
	}
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("ping db (%s): %w", driver, err)
	}
	return &DB{DB: db, Driver: driver}, nil
}

// Q translates ? placeholders to $1, $2, … for PostgreSQL.
// SQLite and other drivers use ? and are returned unchanged.
func (db *DB) Q(query string) string {
	if db.Driver != "postgres" {
		return query
	}
	var b strings.Builder
	n := 0
	for i := 0; i < len(query); i++ {
		if query[i] == '?' {
			n++
			b.WriteString("$" + strconv.Itoa(n))
		} else {
			b.WriteByte(query[i])
		}
	}
	return b.String()
}

// PK returns a cross-driver auto-increment primary key column definition.
func (db *DB) PK() string {
	if db.Driver == "postgres" {
		return "BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY"
	}
	return "INTEGER PRIMARY KEY"
}

// InsertReturningID executes an INSERT and returns the new row's ID.
// query must end with RETURNING id for Postgres, or be a plain INSERT for SQLite.
func (db *DB) InsertReturningID(query string, args ...any) (int64, error) {
	if db.Driver == "postgres" {
		var id int64
		err := db.QueryRow(db.Q(query+" RETURNING id"), args...).Scan(&id)
		return id, err
	}
	res, err := db.Exec(db.Q(query), args...)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// Migrate creates all required tables if they do not already exist.
func Migrate(db *DB) error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS sessions (
			id                 TEXT    PRIMARY KEY,
			username           TEXT    NOT NULL,
			imap_host          TEXT    NOT NULL,
			imap_port          INTEGER NOT NULL,
			smtp_host          TEXT    NOT NULL,
			smtp_port          INTEGER NOT NULL,
			encrypted_password TEXT    NOT NULL,
			created_at         TEXT    NOT NULL,
			expires_at         TEXT    NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS user_settings (
			username  TEXT NOT NULL,
			imap_host TEXT NOT NULL,
			key       TEXT NOT NULL,
			value     TEXT NOT NULL,
			PRIMARY KEY (username, imap_host, key)
		)`,
		`CREATE TABLE IF NOT EXISTS contacts (
			id         ` + db.PK() + `,
			owner      TEXT NOT NULL,
			imap_host  TEXT NOT NULL,
			name       TEXT NOT NULL DEFAULT '',
			notes      TEXT NOT NULL DEFAULT ''
		)`,
		`CREATE TABLE IF NOT EXISTS contact_emails (
			id         ` + db.PK() + `,
			contact_id INTEGER NOT NULL,
			email      TEXT    NOT NULL,
			label      TEXT    NOT NULL DEFAULT '',
			UNIQUE (contact_id, email)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_contacts_owner ON contacts (owner, imap_host)`,
		`CREATE INDEX IF NOT EXISTS idx_contact_emails_email ON contact_emails (email)`,
		`CREATE TABLE IF NOT EXISTS contact_groups (
			id         ` + db.PK() + `,
			owner      TEXT NOT NULL,
			imap_host  TEXT NOT NULL,
			name       TEXT NOT NULL DEFAULT ''
		)`,
		`CREATE TABLE IF NOT EXISTS contact_group_members (
			group_id   INTEGER NOT NULL,
			contact_id INTEGER NOT NULL,
			PRIMARY KEY (group_id, contact_id)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_contact_groups_owner ON contact_groups (owner, imap_host)`,
		`CREATE TABLE IF NOT EXISTS calendar_events (
			id          ` + db.PK() + `,
			owner       TEXT NOT NULL,
			imap_host   TEXT NOT NULL,
			title       TEXT NOT NULL DEFAULT '',
			description TEXT NOT NULL DEFAULT '',
			location    TEXT NOT NULL DEFAULT '',
			all_day     INTEGER NOT NULL DEFAULT 0,
			starts_at   TEXT NOT NULL,
			ends_at     TEXT NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_calendar_events_owner ON calendar_events (owner, imap_host, starts_at)`,
	}
	for _, stmt := range stmts {
		if _, err := db.Exec(stmt); err != nil {
			return fmt.Errorf("migrate: %w", err)
		}
	}

	// Best-effort column additions for schema evolution (ignored if already exist).
	_, _ = db.Exec(`ALTER TABLE calendar_events ADD COLUMN rrule TEXT NOT NULL DEFAULT ''`)
	_, _ = db.Exec(`ALTER TABLE sessions ADD COLUMN user_agent TEXT NOT NULL DEFAULT ''`)
	_, _ = db.Exec(`ALTER TABLE sessions ADD COLUMN last_activity_at TEXT NOT NULL DEFAULT ''`)
	_, _ = db.Exec(`ALTER TABLE contacts ADD COLUMN pgp_public_key TEXT NOT NULL DEFAULT ''`)

	// Message index tables (best-effort; already exist on upgrade).
	for _, s := range []string{
		`CREATE TABLE IF NOT EXISTS message_index (
			username        TEXT    NOT NULL,
			imap_host       TEXT    NOT NULL,
			folder          TEXT    NOT NULL,
			uid             INTEGER NOT NULL,
			subject         TEXT    NOT NULL DEFAULT '',
			from_addr       TEXT    NOT NULL DEFAULT '',
			date            TEXT    NOT NULL DEFAULT '',
			read            INTEGER NOT NULL DEFAULT 0,
			flagged         INTEGER NOT NULL DEFAULT 0,
			has_attachments INTEGER NOT NULL DEFAULT 0,
			size            INTEGER NOT NULL DEFAULT 0,
			message_id      TEXT    NOT NULL DEFAULT '',
			in_reply_to     TEXT    NOT NULL DEFAULT '',
			refs            TEXT    NOT NULL DEFAULT '',
			PRIMARY KEY (username, imap_host, folder, uid)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_message_index_search
			ON message_index (username, imap_host, subject, from_addr)`,
		`CREATE TABLE IF NOT EXISTS folder_index_state (
			username     TEXT    NOT NULL,
			imap_host    TEXT    NOT NULL,
			folder       TEXT    NOT NULL,
			uid_validity INTEGER NOT NULL DEFAULT 0,
			PRIMARY KEY (username, imap_host, folder)
		)`,
	} {
		_, _ = db.Exec(s)
	}

	return nil
}
