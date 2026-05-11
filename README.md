<p align="center">
  <img src="assets/letrvu-logo-stacked.svg" width="120" alt="letrvu" />
</p>

<p align="center">
  A clean, self-hosted webmail client for IMAP servers
</p>

<p align="center">
  <img src="https://img.shields.io/badge/license-MIT-1D9E75" />
  <img src="https://img.shields.io/badge/go-1.26+-00ADD8" />
  <img src="https://img.shields.io/badge/vue-3-42b883" />
  <img src="https://img.shields.io/badge/status-early%20development-orange" />
</p>

---

**letrvu** ("letter view") is a lightweight, modern webmail client that connects to any standard IMAP/SMTP server. No bundled mail server, no PHP.

## Features

- Connects to any IMAP server (Dovecot, Cyrus, Gmail, Fastmail, etc.)
- Three-panel layout: folders → message list → message view
- HTML email rendered in a sandboxed iframe
- Real-time new mail via IMAP IDLE + Server-Sent Events
- Compose, reply, forward, delete, search
- Attachment download
- Address book with vCard import/export and compose autocomplete
- SQLite (default) or PostgreSQL session/settings/contacts storage
- Dark mode
- Ships as a single Go binary

## Stack

| Layer | Technology |
|---|---|
| Backend | Go 1.26 `net/http` |
| IMAP | `emersion/go-imap/v2` |
| SMTP | `emersion/go-smtp` |
| Frontend | Vue 3 + Vite + Pinia |
| Database | SQLite (`modernc.org/sqlite`) or PostgreSQL (`pgx`) |
| Deploy | Single binary or Docker |

## Project structure

```
cmd/letrvu/             main entrypoint
internal/
  api/                  HTTP router + handlers
  imap/                 IMAP client wrapper + IDLE
  smtp/                 outbound mail
  session/              DB-backed session store (AES-256-GCM)
  contacts/             address book store + vCard codec
  settings/             per-user key/value settings
  db/                   database wrapper (SQLite / PostgreSQL)
web/
  src/
    pages/              LoginPage.vue, MailPage.vue, ContactsPage.vue
    components/         FolderList, MessageList, MessageView,
                        ComposeModal, AddressInput, ContactModal
    stores/             auth.js, mail.js, contacts.js  (Pinia)
    composables/        useMailEvents.js (SSE), useDarkMode.js
web/public/assets/      logo files (SVG)
Dockerfile              multi-stage build
```

## Development

### Prerequisites

- Go 1.26+
- Node.js 20+

### Run locally

```bash
# 1. Start the Go backend (port 8080)
go run ./cmd/letrvu

# 2. In a second terminal, start the Vue dev server (port 5173)
cd web
npm install
npm run dev
```

The Vite dev server proxies `/api/*` to `localhost:8080`, so you only visit `http://localhost:5173`.

### Running tests

```bash
# Go backend tests
go test ./...

# Frontend tests (Vitest)
cd web
npm test          # run once
npm run test:watch  # watch mode
```

### Build for production

```bash
cd web && npm run build   # outputs to internal/api/static/
go build -o letrvu ./cmd/letrvu
./letrvu -addr :8080
```

## Docker

```bash
docker build -t letrvu .
docker run -p 8080:8080 letrvu
```

## Configuration

Copy `.env.example` to `.env` and adjust as needed:

| Variable | Default | Description |
|---|---|---|
| `LISTEN_ADDR` | `:8080` | HTTP listen address |
| `DB_DRIVER` | `sqlite` | `sqlite` or `postgres` |
| `DATABASE_URL` | `./letrvu.db` | SQLite path or Postgres DSN |
| `SESSION_SECRET` | *(ephemeral)* | 32-byte hex secret — set this in production |
| `IMAP_HOST` / `IMAP_PORT` | — / `993` | Pre-fill login form |
| `SMTP_HOST` / `SMTP_PORT` | — / `587` | Pre-fill login form |
| `IMAP_INSECURE_TLS` | `true` | Skip TLS cert verification (self-signed certs) |
| `WEBMAIL_HOSTNAME` | `localhost` | Right-hand side of generated `Message-ID` headers |
| `LOGIN_MAX_ATTEMPTS` | `5` | Failed logins per IP before lockout |
| `LOGIN_WINDOW` | `1m` | Sliding window for counting failures |
| `LOGIN_LOCKOUT` | `15m` | Lockout duration after max failures |

## Roadmap

- [x] IMAP folder listing (alphabetical)
- [x] Message list with pagination
- [x] Message view (HTML + plain text, RFC 2047 encoded headers)
- [x] Compose / reply / forward
- [x] Delete + mark read/unread
- [x] IMAP IDLE → SSE push notifications
- [x] Attachments (view + download)
- [x] Search (server-side IMAP SEARCH)
- [x] Embed frontend via `go:embed`
- [x] Dark mode
- [x] DB-backed sessions (SQLite / PostgreSQL)
- [x] Per-user settings (display name, signature)
- [x] Address book with vCard import/export
- [x] Compose autocomplete from address book
- [x] Calendar (month + week view, add/edit/delete events)
- [x] iCal import/export
- [x] Email invite detection ("Add to calendar" button)
- [X] Signature insertion in compose
- [x] Save sent messages to Sent folder (IMAP APPEND)
- [x] Draft saving (IMAP APPEND to Drafts)
- [x] Reply-all
- [x] IMAP folder subscription handling
- [ ] Calendar recurring events (RRULE)
- [ ] Calendar outgoing invites (attach iCal to composed email)
- [ ] Multi-account support
- [X] Move messages between folders
- [X] Show message source
- [X] Flag messages
- [X] Multiple identities
- [X] Attachment preview
- [X] Brute force login protection
- [x] Folder management (create / rename / delete IMAP folders)
- [x] Bulk actions (select multiple messages → delete / move / mark read)
- [x] Keyboard shortcuts (n/p next/prev, r reply, d delete, c compose)
- [ ] HTML compose (rich text editor)
- [ ] Conversation / thread view
- [ ] Unread count in browser tab title
- [x] Desktop notifications (Browser Notification API + IMAP IDLE)
- [ ] Mobile-responsive layout
- [x] Mark as spam (move to Junk folder)
- [ ] Cross-folder search
- [ ] Undo send (configurable delay before SMTP submission)
- [ ] Vacation / autoresponder (Sieve)
- [ ] Contact groups / distribution lists
- [ ] Per-sender image trust ("always show images from this sender")
- [ ] Print view
- [ ] PGP / S-MIME encryption

## Keyboard shortcuts

| Key | Action |
|-----|--------|
| `c` | Compose new message |
| `r` | Reply to current message |
| `n` | Next message (older) |
| `p` | Previous message (newer) |
| `d` | Delete current message |
| `Esc` | Close modal / overlay (compose, attachment preview, message source) |

Shortcuts are disabled when focus is inside a text field or the compose window is open.

## License

MIT
