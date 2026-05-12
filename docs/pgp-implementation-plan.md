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

- **Private keys must never touch the server.** The browser generates, holds, and uses the private key. This is non-negotiable — violating it defeats the purpose of the feature.
- Encryption of outbound mail is only possible when all recipients have a stored public key.

## Suggested v1 scope

### 1. Key management (Settings)
- Generate a new key pair in the browser (name + email pre-filled from account settings)
- Import an existing private key (armored `.asc` / `.pgp` file)
- Export the private key and the public key separately
- Private key stored in browser `localStorage`; never sent to the server

### 2. Per-contact public key storage
- Add a `pgp_public_key TEXT` column to the `contacts` table
- UI in the contact detail view: paste or import a public key, display fingerprint

### 3. Compose integration
- Sign toggle: always available when the user has a private key loaded
- Encrypt toggle: available only when every recipient has a stored public key; grayed out otherwise with a tooltip explaining why
- On send: sign and/or encrypt using `openpgp.js` before the MIME message is assembled

### 4. Message view integration
- Detect PGP inline (`-----BEGIN PGP MESSAGE-----`) and PGP/MIME (`multipart/encrypted`)
- Decrypt if the user's private key is loaded
- Verify signature and show a clear verified / failed / unknown-key badge

## Key library

[`openpgp.js`](https://openpgpjs.org/) — mature, actively maintained, runs in the browser with no server dependency.

## Out of scope for v1

- Key server lookup (WKD, keys.openpgp.org)
- Key signing / web of trust UI
- S/MIME support
- Per-session key passphrase prompts (can add later)
