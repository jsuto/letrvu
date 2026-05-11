package main

import (
	"crypto/rand"
	"crypto/tls"
	"encoding/hex"
	"flag"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"github.com/jsuto/letrvu/internal/api"
	"github.com/jsuto/letrvu/internal/calendar"
	"github.com/jsuto/letrvu/internal/contacts"
	"github.com/jsuto/letrvu/internal/db"
	"github.com/jsuto/letrvu/internal/imap"
	"github.com/jsuto/letrvu/internal/session"
	"github.com/jsuto/letrvu/internal/settings"
	"github.com/jsuto/letrvu/internal/smtp"
)

func main() {
	godotenv.Load()

	addr := flag.String("addr", envOr("LISTEN_ADDR", ":8080"), "listen address")
	flag.Parse()

	insecureTLS := envBool("IMAP_INSECURE_TLS", true)
	imap.DefaultTLSConfig = &tls.Config{
		InsecureSkipVerify: insecureTLS, //nolint:gosec
	}
	smtp.DefaultTLSConfig = &tls.Config{
		InsecureSkipVerify: insecureTLS, //nolint:gosec
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
	if strings.EqualFold(os.Getenv("LOG_LEVEL"), "debug") {
		imap.Debug = true
		log.Println("debug logging enabled")
	}

	cfg := api.ServerConfig{
		IMAPHost:        envOr("IMAP_HOST", ""),
		IMAPPort:        envInt("IMAP_PORT", 993),
		SMTPHost:        envOr("SMTP_HOST", ""),
		SMTPPort:        envInt("SMTP_PORT", 587),
		SecureCookies:   envBool("SECURE_COOKIES", false),
		TrustedProxy:    envCIDR("TRUSTED_PROXY"),
		FolderCacheTTL:  envDuration("FOLDER_CACHE_TTL", 2*time.Minute),
		InternalDomains: envDomains("INTERNAL_DOMAINS"),
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

// envDomains parses a comma-separated list of domain names from an env var.
// e.g. INTERNAL_DOMAINS=example.com,example.org
func envDomains(key string) []string {
	v := os.Getenv(key)
	if v == "" {
		return nil
	}
	var out []string
	for _, d := range strings.Split(v, ",") {
		if d = strings.TrimSpace(strings.ToLower(d)); d != "" {
			out = append(out, d)
		}
	}
	return out
}

// envCIDR parses an IP address or CIDR range from an env var.
// A plain IP (e.g. "127.0.0.1") is treated as a /32 (or /128 for IPv6).
// Returns nil if the variable is unset or empty.
func envCIDR(key string) *net.IPNet {
	v := os.Getenv(key)
	if v == "" {
		return nil
	}
	if !strings.Contains(v, "/") {
		// Plain IP — append host mask so ParseCIDR accepts it.
		if strings.Contains(v, ":") {
			v += "/128" // IPv6
		} else {
			v += "/32" // IPv4
		}
	}
	_, cidr, err := net.ParseCIDR(v)
	if err != nil {
		log.Fatalf("TRUSTED_PROXY: invalid IP/CIDR %q: %v", os.Getenv(key), err)
	}
	return cidr
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

func envDuration(key string, fallback time.Duration) time.Duration {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		log.Printf("WARNING: invalid %s=%q, using default %s", key, v, fallback)
		return fallback
	}
	return d
}
