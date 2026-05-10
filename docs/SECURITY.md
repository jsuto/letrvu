# Security

This document describes the security threats relevant to a webmail client and the mitigations in place or planned for letrvu.

## 1. Tracking pixels / remote image loading

**Threat:** Senders embed 1×1 transparent images. Loading them confirms the recipient's email address is live, leaks their IP address, approximate geolocation, mail client, and read timestamp to the sender's server.

**Mitigation (implemented):** Remote images (`http://`, `https://`, and protocol-relative `//` URLs in `<img src>`, inline `style` attributes, and `<style>` blocks) are replaced with a local placeholder before the HTML is rendered. A banner informs the user and offers a one-click opt-in to load images for that message.

## 2. XSS via HTML email content

**Threat:** Malicious HTML in an email body (`<script>`, inline event handlers like `onerror=`, `javascript:` hrefs, CSS `expression()`) executes code in the reader's browser session.

**Mitigation (partial):** HTML email is rendered inside a sandboxed `<iframe srcdoc="...">`. The current sandbox value is `allow-same-origin`, which is too permissive — it allows same-origin scripts to run. The safest value is a bare `sandbox` attribute (no tokens), which blocks all script execution inside the frame. `allow-scripts` must never be added.

As defense-in-depth, a server-side HTML sanitizer (e.g., [bluemonday](https://github.com/microcosm-cc/bluemonday)) should be run on the HTML body before it is sent to the client, stripping any dangerous tags and attributes before they even reach the iframe.

**Recommended fix:** Change `sandbox="allow-same-origin"` to `sandbox` in `MessageView.vue`.

## 3. Phishing via link spoofing

**Threat:** An email displays a trusted domain as the link text while the `href` points to a different, malicious destination. Users click without inspecting the URL.

**Mitigation (not yet implemented):** The browser shows the real `href` in the status bar on hover. A stronger mitigation is a link-interception layer in the message view that compares the visible link text to the `href` domain and surfaces a warning when they differ.

## 4. Session credential storage

**Threat:** IMAP/SMTP credentials stored server-side can be exfiltrated if the database is compromised.

**Mitigation (in place):** Credentials are encrypted with AES-256-GCM. The encryption key is derived from a session secret that lives only in the user's browser cookie — a database dump alone is insufficient to recover credentials.

## 5. Cookie / session hijacking

**Threat:** A stolen session cookie grants full account access, including the ability to read, send, and delete mail.

**Mitigation:** Session cookies must be set with `HttpOnly`, `Secure`, and `SameSite=Strict`. Audit `internal/session/` to confirm all three flags are applied.

## 6. CSRF

**Threat:** A malicious third-party site triggers a state-changing API call (send message, delete message) on behalf of a logged-in user whose cookie is sent automatically by the browser.

**Mitigation:** `SameSite=Strict` on session cookies prevents the cookie from being sent on cross-site requests, which covers most CSRF scenarios. A CSRF token on mutating endpoints provides belt-and-suspenders protection.

## 7. IMAP/SMTP TLS verification

**Threat:** The `IMAP_INSECURE_TLS=true` environment variable disables TLS certificate validation. On a real mail server this exposes credentials and message content to network interception.

**Mitigation:** This flag must never be set in production. The UI should display a visible warning when the server connection was established without certificate verification.

## 8. Content-Security-Policy for the app shell

**Threat:** Even with email-body isolation, reflected or stored XSS in the Vue application itself (e.g., via subject lines, sender names, or folder names rendered without sanitization) can execute code.

**Mitigation (not yet implemented):** Add a strict `Content-Security-Policy` response header in the Go server:

```
Content-Security-Policy: default-src 'self'; script-src 'self'; object-src 'none'; frame-ancestors 'none'
```

The `srcdoc` iframe is governed by its own sandbox attribute, not the parent page's CSP, so this header does not conflict with HTML email rendering.

---

## Priority summary

| Priority | Item |
|---|---|
| High | Change `sandbox="allow-same-origin"` to `sandbox` in `MessageView.vue` |
| High | Audit session cookie flags (`HttpOnly`, `Secure`, `SameSite=Strict`) |
| Medium | Add `Content-Security-Policy` header in the Go HTTP server |
| Medium | Server-side HTML sanitization (bluemonday) before serving message bodies |
| Low | Link destination warning for mismatched href text and href URL |
| Low | Per-sender "always show images" preference persisted in settings |
