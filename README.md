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
  <img src="https://img.shields.io/badge/status-beta-blue" />
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

### Docker Compose (with PostgreSQL)

```bash
cp .env.example .env
# edit .env: set SESSION_SECRET, POSTGRES_PASSWORD, IMAP_HOST, SMTP_HOST

# Create the external volume once — persists even if the stack is removed
docker volume create db_data

docker compose up -d
```

The `db_data` volume is declared external so that `docker compose down` (or even `docker compose down -v`) cannot accidentally delete your database. Only `docker volume rm db_data` will remove it.

## Production deployment

letrvu speaks plain HTTP and must sit behind a TLS-terminating reverse proxy in production. Set `SECURE_COOKIES=true` in your `.env` once HTTPS is in place — this adds the `Secure` flag to session and CSRF cookies so they are never sent over plain HTTP.

### Traefik (recommended for Docker Compose)

Traefik integrates directly with Docker Compose via container labels — no separate config file needed. Add a Traefik service to your `docker-compose.yml` and annotate the `letrvu` service with routing labels:

```yaml
services:
  traefik:
    image: traefik:latest
    command:
      - --providers.docker=true
      - --providers.docker.exposedbydefault=false
      - --entrypoints.web.address=:80
      - --entrypoints.web.http.redirections.entrypoint.to=websecure
      - --entrypoints.websecure.address=:443
      - --certificatesresolvers.le.acme.tlschallenge=true
      - --certificatesresolvers.le.acme.email=you@example.com
      - --certificatesresolvers.le.acme.storage=/acme/acme.json
    ports:
      - "80:80"
      - "443:443"
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock:ro
      - acme_data:/acme
    restart: unless-stopped

  letrvu:
    image: sutoj/letrvu:latest
    labels:
      - traefik.enable=true
      - traefik.http.routers.letrvu.rule=Host(`mail.example.com`)
      - traefik.http.routers.letrvu.entrypoints=websecure
      - traefik.http.routers.letrvu.tls.certresolver=le
      # Required for SSE (real-time new mail push)
      - traefik.http.services.letrvu.loadbalancer.responseforwarding.flushinterval=-1
    # ... rest of letrvu service config

volumes:
  acme_data:
    external: true
```

Create the ACME volume before first run:
```bash
docker volume create acme_data
```

Set `TRUSTED_PROXY` to Traefik's container IP or the Docker bridge subnet (e.g. `172.17.0.0/16`) so letrvu reads the real client IP from `X-Forwarded-For`:
```bash
TRUSTED_PROXY=172.17.0.0/16
```

### Caddy (recommended for bare-metal)

Caddy obtains and renews Let's Encrypt certificates automatically with zero config.

```
# /etc/caddy/Caddyfile
mail.example.com {
    reverse_proxy localhost:8080
}
```

```bash
sudo systemctl reload caddy
```

### nginx

```nginx
# /etc/nginx/sites-available/letrvu
server {
    listen 80;
    server_name mail.example.com;
    return 301 https://$host$request_uri;
}

server {
    listen 443 ssl;
    server_name mail.example.com;

    ssl_certificate     /etc/letsencrypt/live/mail.example.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/mail.example.com/privkey.pem;

    # Required for SSE (real-time new mail push)
    proxy_buffering off;
    proxy_read_timeout 3600s;

    location / {
        proxy_pass http://localhost:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

```bash
sudo certbot --nginx -d mail.example.com
sudo systemctl reload nginx
```

Set `TRUSTED_PROXY=127.0.0.1` in `.env` so letrvu reads the real client IP from `X-Forwarded-For` for accurate audit logs and brute-force protection.

### `.env` settings for production

```bash
SESSION_SECRET=$(openssl rand -hex 32)
POSTGRES_PASSWORD=$(openssl rand -hex 16)
SECURE_COOKIES=true
WEBMAIL_HOSTNAME=mail.example.com
TRUSTED_PROXY=127.0.0.1        # or 172.17.0.0/16 when using Traefik
IMAP_INSECURE_TLS=false        # only if your mail server has a valid certificate
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
- [x] Calendar recurring events (RRULE)
- [x] Calendar outgoing invites (attach iCal to composed email)
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
- [x] HTML compose (rich text editor)
- [x] Conversation / thread view
- [x] Unread count in browser tab title
- [x] Desktop notifications (Browser Notification API + IMAP IDLE)
- [x] Mobile-responsive layout
- [x] Mark as spam (move to Junk folder)
- [X] Cross-folder search
- [ ] Undo send (configurable delay before SMTP submission)
- [ ] Vacation / autoresponder (Sieve)
- [x] Contact groups / distribution lists
- [ ] Per-sender image trust ("always show images from this sender")
- [x] Print view
- [ ] PGP / S-MIME encryption
- [x] Docker scout scanning
- [x] Session timeout / logout all devices
- [x] Spam flag feedback (`$Junk` IMAP flag)

## Releasing

1. Bump the version in `VERSION`:
   ```bash
   echo "0.2" > VERSION
   git add VERSION
   git commit -m "Release v0.2"
   ```
2. Tag and push — this triggers the release workflow:
   ```bash
   git tag v0.2
   git push origin master --tags
   ```

The workflow will build Linux binaries (`amd64`, `arm64`), run a Docker Scout CVE scan, push a multi-platform Docker image (`sutoj/letrvu:<version>` and `sutoj/letrvu:latest`), and create a GitHub Release with the binaries and a `sha256sums.txt` attached.

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
