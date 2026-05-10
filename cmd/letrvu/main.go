package main

import (
	"crypto/rand"
	"crypto/tls"
	"encoding/hex"
	"flag"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/joho/godotenv"
	"github.com/jsuto/letrvu/internal/api"
	"github.com/jsuto/letrvu/internal/calendar"
	"github.com/jsuto/letrvu/internal/contacts"
	"github.com/jsuto/letrvu/internal/db"
	"github.com/jsuto/letrvu/internal/imap"
	"github.com/jsuto/letrvu/internal/session"
	"github.com/jsuto/letrvu/internal/settings"
)

func main() {
	godotenv.Load()

	addr := flag.String("addr", envOr("LISTEN_ADDR", ":8080"), "listen address")
	flag.Parse()

	imap.DefaultTLSConfig = &tls.Config{
		InsecureSkipVerify: envBool("IMAP_INSECURE_TLS", true), //nolint:gosec
	}

	// Database
	database, err := db.Open(
		envOr("DB_DRIVER", "sqlite"),
		envOr("DATABASE_URL", "./letrvu.db"),
	)
	if err != nil {
		log.Fatalf("database: %v", err)
	}
	if err := db.Migrate(database); err != nil {
		log.Fatalf("migrate: %v", err)
	}

	// Session secret — compound key scheme: server secret + per-client nonce.
	secret := loadOrGenerateSecret()

	// Stores
	sessions := session.NewStore(database, secret)
	sessions.DeleteExpired()

	settingsStore := settings.NewStore(database)
	contactsStore := contacts.NewStore(database)
	calendarStore := calendar.NewStore(database)

	// Server-level IMAP/SMTP defaults (pre-fill login form via /api/config).
	cfg := api.ServerConfig{
		IMAPHost:      envOr("IMAP_HOST", ""),
		IMAPPort:      envInt("IMAP_PORT", 993),
		SMTPHost:      envOr("SMTP_HOST", ""),
		SMTPPort:      envInt("SMTP_PORT", 587),
		SecureCookies: envBool("SECURE_COOKIES", false),
	}

	handler := api.NewRouter(sessions, settingsStore, contactsStore, calendarStore, cfg)

	log.Printf("letrvu listening on %s", *addr)
	if err := http.ListenAndServe(*addr, handler); err != nil {
		log.Fatal(err)
	}
}

// loadOrGenerateSecret parses SESSION_SECRET from the environment.
// If unset, an ephemeral secret is generated and the operator is warned that
// sessions will not survive a server restart.
func loadOrGenerateSecret() []byte {
	h := os.Getenv("SESSION_SECRET")
	if h != "" {
		b, err := hex.DecodeString(h)
		if err != nil {
			log.Fatalf("SESSION_SECRET must be hex-encoded: %v", err)
		}
		if len(b) < 32 {
			log.Fatal("SESSION_SECRET must be at least 32 bytes (64 hex chars)")
		}
		return b
	}
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		log.Fatalf("generate secret: %v", err)
	}
	log.Printf("WARNING: SESSION_SECRET not set — sessions will not survive restart.")
	log.Printf("Add to .env:  SESSION_SECRET=%s", hex.EncodeToString(b))
	return b
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func envBool(key string, fallback bool) bool {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return fallback
	}
	return b
}

func envInt(key string, fallback int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return n
}
