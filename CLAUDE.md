# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What This Is

**Letrvu** is a self-hosted webmail client that connects to standard IMAP/SMTP servers. It ships as a single Go binary with an embedded Vue 3 SPA. It does **not** bundle a mail server.

## Development Commands

### Backend (Go, port 8080)
```bash
go run ./cmd/letrvu
```

### Frontend (Vue 3 + Vite, port 5173 — proxies `/api` to `:8080`)
```bash
cd web && npm install && npm run dev
```

### Production Build
```bash
cd web && npm run build   # outputs to internal/api/static/
go build -o letrvu ./cmd/letrvu
./letrvu -addr :8080
```

### Docker
```bash
docker build -t letrvu .
docker run -p 8080:8080 letrvu
```

## Configuration

Copy `.env.example` to `.env`. Key variables:
- `DB_DRIVER` — `sqlite` (default) or `postgres`
- `DATABASE_URL` — SQLite path or PostgreSQL DSN
- `SESSION_SECRET` — 32-byte hex string (required in production)
- `IMAP_HOST`, `IMAP_PORT`, `SMTP_HOST`, `SMTP_PORT` — pre-fill login form
- `IMAP_INSECURE_TLS=true` — skip TLS cert validation (for self-signed certs)

## Architecture

```
Frontend (Vue 3 SPA)
  └─ /api/* (HTTP REST + SSE)
       └─ Go HTTP server (net/http.ServeMux)
            ├─ internal/api/      — router + all HTTP handlers
            ├─ internal/imap/     — IMAP client (go-imap/v2), IDLE for push
            ├─ internal/smtp/     — outbound mail composition
            ├─ internal/session/  — AES-256-GCM encrypted credentials in DB
            ├─ internal/contacts/ — address book (vCard import/export)
            ├─ internal/calendar/ — events (iCal import/export)
            ├─ internal/settings/ — per-user key/value prefs
            ├─ internal/db/       — database abstraction (SQLite / PostgreSQL)
            └─ internal/mime/     — MIME parsing utilities
```

**Key data flow:**
1. Login → user credentials AES-256-GCM encrypted and stored per session in DB
2. Each request retrieves credentials from session and opens IMAP/SMTP connections
3. IMAP IDLE monitors for new mail → server-sent events (SSE) pushed to frontend via `/api/events`
4. Contacts and calendar data live in local DB (independent of mail server)

**Frontend (`web/src/`):**
- `pages/` — LoginPage, MailPage, ContactsPage, CalendarPage
- `components/` — FolderList, MessageList, MessageView, Compose, etc.
- `stores/` — Pinia stores: `auth`, `mail`, `contacts`, `calendar`
- `composables/useMailEvents.js` — SSE connection for real-time new-mail notifications

**Key dependencies:**
- Go: `github.com/emersion/go-imap/v2`, `go-smtp`, `go-message`, `go-vcard`, `go-ical`, `modernc.org/sqlite`, `pgx/v5`
- Frontend: Vue 3, Vue Router 4, Pinia, Vite 5
