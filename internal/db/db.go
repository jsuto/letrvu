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
	}
	for _, stmt := range stmts {
		if _, err := db.Exec(stmt); err != nil {
			return fmt.Errorf("migrate: %w", err)
		}
	}
	return nil
}
