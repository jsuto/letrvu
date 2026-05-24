# i18n Implementation Plan

## Scope

~800–1,200 user-facing strings across 21 Vue files. No existing i18n infrastructure.
English is the only shipped locale. The scaffold is designed so additional languages can
be added later by dropping a translation file into `web/src/i18n/locales/`.

Out of scope (for now): RTL layout support, pluralization helpers beyond what vue-i18n
provides automatically, backend string translation.

---

## Phase 1 — Infrastructure

### 1. Install vue-i18n v9

```bash
cd web && npm install vue-i18n@9
```

### 2. Create the i18n module

```
web/src/i18n/
  index.js          ← creates and exports the i18n instance
  locales/
    en.json         ← all English strings, organized by namespace
```

`index.js`:
```js
import { createI18n } from 'vue-i18n'
import en from './locales/en.json'

const savedLocale = localStorage.getItem('locale') || 'en'

export default createI18n({
  legacy: false,        // use Composition API mode
  locale: savedLocale,
  fallbackLocale: 'en',
  messages: { en },
})
```

### 3. Register in main.js

```js
import i18n from './i18n/index.js'
app.use(i18n)
```

### 4. Add locale to the settings store

`stores/settings.js` — add a computed:
```js
const locale = computed(() => settings.value.locale || 'en')
```

Watch it in `App.vue` to keep vue-i18n in sync after the settings load from the API:
```js
watch(locale, val => { i18n.global.locale.value = val })
```

Backend: add `"locale": true` to the `allowed` map in `internal/settings/store.go`.

### 5. Add language selector to SettingsModal.vue

A `<select>` bound to `form.locale` using the same pattern as the Undo send selector.
For now it only lists English — more options are added here as new locale files land.

---

## Phase 2 — String extraction

Work file by file. Replace every hardcoded template string with a `$t()` call and add
the key to `en.json`.

**Template strings:**
```html
<!-- before -->
<button>Send</button>
<input placeholder="Search…" />

<!-- after -->
<button>{{ $t('compose.send') }}</button>
<input :placeholder="$t('messageList.searchPlaceholder')" />
```

**Script strings** (inside `<script setup>`):
```js
const { t } = useI18n()
const label = computed(() => t('folders.inbox'))
```

### Key namespace structure

```
common.*          Save, Cancel, Delete, Close, Loading…, Back, Edit
login.*           IMAP server, Email address, Password, Sign in, Verify code
compose.*         New message, From, To, CC, BCC, Subject, Send, Save draft,
                  Attach file, Write your message…
messageList.*     Search placeholder, Mark read, Mark unread, Archive, Spam,
                  Not spam, Delete, Select all, N messages selected
messageView.*     Reply, Reply all, Forward, Delete, Archive, Spam, Not spam,
                  Print, Show source, Unsubscribe, Attachments, To, Loading…
threadView.*      N messages (in thread header), Archive, Reply, Reply all
folders.*         Inbox, Sent, Drafts, Trash, Junk, Archive, Manage folders,
                  Create folder, Rename, Delete folder, Quota labels
settings.*        All SettingsModal labels — display name, signature,
                  undo send, poll interval, notifications, 2FA, PGP,
                  vacation responder, read receipts, trusted senders,
                  message templates, identities (~80 strings)
contacts.*        All ContactsPage + ContactModal strings
calendar.*        All CalendarPage + EventModal strings
filters.*         FiltersModal strings
templates.*       TemplatesModal strings
errors.*          API / network error messages
```

### Recommended extraction order

1. `common` — unblocks every other file
2. `ComposeModal`, `MessageView`, `MessageList` — core mail workflow
3. `SettingsModal` — most strings in one file
4. `LoginPage`
5. Remaining components and pages

---

## Phase 3 — Pluralization and date formatting

vue-i18n v9 handles both natively.

**Pluralization** (e.g. thread message count):
```json
{ "threadView": { "messageCount": "no messages | 1 message | {n} messages" } }
```
```html
{{ $tc('threadView.messageCount', thread.messages.length) }}
```

**Date/time formatting** via `$d()` — replaces the scattered `toLocaleString()` calls in
`ThreadView`, `MessageView`, `MessageList`, etc. Dates render in the user's locale
automatically once a second language is added. Define named formats in `index.js`:
```js
datetimeFormats: {
  en: {
    short:  { month: 'short', day: 'numeric' },
    long:   { year: 'numeric', month: 'short', day: 'numeric',
              hour: '2-digit', minute: '2-digit' },
  }
}
```

---

## Phase 4 — Adding a second language (future)

1. Duplicate `en.json` → `de.json` (or target language).
2. Translate all values. vue-i18n falls back to `en` for any missing key, so partial
   translations are safe to ship.
3. Add the locale to the `<select>` in `SettingsModal.vue` and import the messages in
   `index.js`.

---

## What does NOT need i18n

- IMAP folder names passed to the backend (INBOX, Sent, etc.)
- API paths, MIME types, IMAP flags
- Date/time strings sent to or received from the server (always RFC 3339)
- Internal Pinia store keys and settings store keys
