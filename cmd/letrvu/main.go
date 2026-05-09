package main

import (
	"crypto/tls"
	"flag"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/joho/godotenv"
	"github.com/yourusername/letrvu/internal/api"
	"github.com/yourusername/letrvu/internal/imap"
	"github.com/yourusername/letrvu/internal/session"
)

func main() {
	// Load .env if present; ignore error when the file doesn't exist.
	godotenv.Load()

	addr := flag.String("addr", envOr("LISTEN_ADDR", ":8080"), "listen address")
	flag.Parse()

	imap.DefaultTLSConfig = &tls.Config{
		InsecureSkipVerify: envBool("IMAP_INSECURE_TLS", true), //nolint:gosec
	}

	sessions := session.NewStore()
	handler := api.NewRouter(sessions)

	log.Printf("letrvu listening on %s", *addr)
	if err := http.ListenAndServe(*addr, handler); err != nil {
		log.Fatal(err)
	}
}

// envOr returns the value of the named environment variable, or fallback if unset.
func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// envBool parses a boolean environment variable, returning fallback if unset or unparseable.
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
