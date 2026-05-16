# PGP Encryption — Implementation Plan

## Decision: PGP over S/MIME

S/MIME requires a CA-issued certificate, which means either the operator runs a CA (complex) or each user obtains their own cert (friction, often costs money). It is practical only in managed corporate deployments and is not a good fit for an open-source self-hosted tool.

PGP has no such dependency. Any user can generate a key pair, and `openpgp.js` runs entirely in the browser — no server-side changes required for the core feature.

## Why it's worth building

- Mail hosting companies deploying Letrvu gain a differentiator — many smaller hosts want to offer PGP but lack the UI to do it
- Privacy-conscious users specifically seek webmail clients with PGP; it is a common reason people self-host
- Sign-only mode (no encryption required from recipient) is useful on its own: it proves authorship on every outbound message with zero coordination needed
- OpenPGP is increasingly mainstream, normalised by services like Proton Mail

## Hard constraints

- **Private key plaintext must never touch the server.** The browser encrypts the key with a passphrase before it is sent anywhere. The server stores only the encrypted blob. This is non-negotiable — violating it defeats the purpose of the feature.
- Encryption of outbound mail is only possible when all recipients have a stored public key.

## Key storage design

### Why not localStorage

localStorage is scoped to a single browser profile. A different browser on the same machine, a reinstall, or "clear site data" all silently destroy the key. This is unacceptable for something as critical as a private key.

### Server-stored encrypted blob

The private key is encrypted in the browser with a **dedicated PGP passphrase** (separate from the login password) before being sent to the server. The server stores an opaque ciphertext it cannot read. On any device, the user enters their passphrase to unlock the key for the session.

**Why a separate passphrase (not the login password):**

If the private key were encrypted with the login password, a password change would invalidate the stored key — requiring the old password to rotate the encryption. The cleaner alternative (KEK — a random key-encryption key, re-encrypted on password change) works well but requires intercepting the login flow client-side and adds meaningful complexity. That is deferred to v2.

A dedicated PGP passphrase is independent of the login password, so password changes never affect it. PGP users expect a separate passphrase; this is standard UX in desktop clients too.

**Session caching:** after the user enters their passphrase and the key is decrypted, the plaintext key is held in memory (a Vue ref / Pinia store) for the duration of the browser session. It is never written to localStorage. The user re-enters the passphrase on next page load (or after a configurable idle timeout).

**Encryption scheme:** AES-256-GCM with a key derived via PBKDF2-SHA256 (310 000 iterations, random 16-byte salt). Salt and IV are stored alongside the ciphertext. All crypto runs in the browser via the Web Crypto API before the blob is sent to the server.

### Password change resilience

Because the passphrase is independent of the login password, no special handling is needed on password change. If a KEK scheme is added in v2, the rotation must be done atomically: decrypt KEK with old password, re-encrypt with new password, save — the private key blob is never touched.

## Suggested v1 scope

### 1. Key management (Settings → Security)

- Generate a new key pair in the browser (name + email pre-filled from account settings)
- Import an existing private key (armored `.asc` / `.pgp` file)
- Export the private key (armored) and the public key separately
- Encrypted private key blob stored in DB via `/api/pgp/key` (POST to save, GET to load)
- Public key fingerprint displayed after unlock
- "Forget key" button (clears the in-memory key for the session; does not delete from server)
- "Delete key" button (removes from server permanently)

### 2. Per-contact public key storage

- Add a `pgp_public_key TEXT` column to the `contacts` table
- UI in the contact detail view: paste or import a `.asc` public key, display fingerprint + key ID
- **WKD auto-lookup:** when composing, attempt to fetch the recipient's public key via WKD (`https://HOST/.well-known/openpgpkey/...`) before marking the Encrypt toggle as unavailable. This requires no extra UI and dramatically improves the encrypt flow.

### 3. Compose integration

- **Sign toggle:** always available when the user has a private key loaded in memory
- **Encrypt toggle:** available only when every recipient has a stored or WKD-fetched public key; grayed out otherwise with a tooltip listing which recipients are missing keys
- On send: sign and/or encrypt using `openpgp.js` before the MIME message is assembled
- **Auto-attach public key:** opt-in setting — attaches the user's armored public key as `publickey.asc` to every signed message, making it easy for recipients to import it

### 4. Message view integration

- Detect PGP/MIME (`multipart/signed`, `multipart/encrypted`) and inline PGP (`-----BEGIN PGP MESSAGE-----`)
- Decrypt if the private key is loaded in memory; prompt for passphrase if not
- Verify signature and show a clear badge: ✓ Verified / ✗ Bad signature / ? Unknown key
- "Import sender's key" button when a public key attachment is present

## Signature verification by recipients

Recipients verify a signed message using the sender's public key. They obtain it via:

1. **Attached public key** — if auto-attach is enabled, the key arrives with the email. Easy, but only as trustworthy as the email channel itself.
2. **WKD / keyserver** — the gold standard. Publishing to `keys.openpgp.org` or hosting a WKD record on your domain lets any OpenPGP client (Thunderbird, Kleopatra, etc.) fetch and verify the key automatically without any manual step.
3. **Out-of-band exchange** — the sender shares the public key fingerprint over a separate channel (Signal, in person) for maximum assurance.

PGP/MIME format (`multipart/signed`) is preferred over inline PGP: the email body is plain readable MIME, so non-PGP clients see a normal email with a small `signature.asc` attachment they can ignore.

## Key library

[`openpgp.js`](https://openpgpjs.org/) — mature, actively maintained, runs in the browser with no server dependency.

## Backend changes

- `internal/settings`: new `GetPGPKey(username) (string, error)` / `SetPGPKey(username, blob string) error` / `DeletePGPKey(username) error` — stores the encrypted blob in the existing key/value settings table
- `internal/api`: new routes `GET /api/pgp/key`, `POST /api/pgp/key`, `DELETE /api/pgp/key`
- `internal/contacts`: add `pgp_public_key TEXT` column (migration)

## Deferred to v2

- Key-encryption key (KEK) scheme tied to login password (deferred to v2)
- Key signing / web of trust UI
- S/MIME support
- Multiple keys per identity
- Per-session passphrase timeout configuration (always re-prompt on page load in v1)
