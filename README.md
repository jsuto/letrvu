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

**letrvu** (from French *lettre vue* — "letter seen") is a lightweight, modern webmail client that connects to any standard IMAP/SMTP server. No bundled mail server, no PHP, no database required.

## Features

- Connects to any IMAP server (Dovecot, Cyrus, Gmail, Fastmail, etc.)
- Three-panel layout: folders → message list → message view
- HTML email rendered in a sandboxed iframe
- Real-time new mail via IMAP IDLE + Server-Sent Events
- Compose, reply, forward, delete
- Ships as a single Go binary

## Stack

| Layer | Technology |
|---|---|
| Backend | Go 1.22 `net/http` |
| IMAP | `emersion/go-imap/v2` |
| SMTP | `emersion/go-smtp` |
| Frontend | Vue 3 + Vite + Pinia |
| Deploy | Single binary or Docker |

## Project structure

```
cmd/letrvu/             main entrypoint
internal/
  api/                  HTTP router + handlers
  imap/                 IMAP client wrapper
  smtp/                 outbound mail
  session/              in-memory session store
web/
  src/
    pages/              LoginPage.vue, MailPage.vue
    components/         FolderList, MessageList, MessageView, ComposeModal
    stores/             auth.js, mail.js  (Pinia)
    composables/        useMailEvents.js  (SSE)
assets/                 logo files (SVG)
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

## Roadmap

- [ ] IMAP folder listing
- [ ] Message list with pagination
- [ ] Message view (HTML + plain text)
- [ ] Compose / reply / forward
- [ ] Delete + mark read/unread
- [ ] IMAP IDLE → SSE push notifications
- [ ] Attachments (view + download)
- [ ] Search (server-side IMAP SEARCH)
- [ ] Embed frontend via `go:embed`
- [ ] Multi-account support
- [ ] Dark mode

## License

MIT
