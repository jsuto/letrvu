# Security

This document describes the security threats relevant to a webmail client and the mitigations in place or planned for letrvu.

## 1. Tracking pixels / remote image loading

**Threat:** Senders embed 1×1 transparent images. Loading them confirms the recipient's email address is live, leaks their IP address, approximate geolocation, mail client, and read timestamp to the sender's server.

**Mitigation (implemented):** Remote images (`http://`, `https://`, and protocol-relative `//` URLs in `<img src>`, inline `style` attributes, and `<style>` blocks) are replaced with a local placeholder before the HTML is rendered. A banner informs the user and offers a one-click opt-in to load images for that message.

## 2. XSS via HTML email content

**Threat:** Malicious HTML in an email body (`<script>`, inline event handlers like `onerror=`, `javascript:` hrefs, CSS `expression()`) executes code in the reader's browser session.

**Mitigation (implemented):** HTML email is rendered inside a sandboxed `<iframe srcdoc="...">` with `sandbox="allow-popups"`. This blocks script execution, same-origin access, and form submission while allowing user-initiated links to open in new tabs. `allow-scripts` must never be added.

As defense-in-depth, [DOMPurify](https://github.com/cure53/DOMPurify) sanitizes the HTML in the frontend before it is set as `srcdoc`. DOMPurify runs in the browser using the same HTML parser that will render the output, which eliminates mutation XSS (mXSS) attacks where a server-side sanitizer and the browser would parse the same markup differently. It strips script tags, `javascript:` URLs, inline event handlers, and other dangerous constructs.

## 3. Phishing via link spoofing

**Threat:** An email displays a trusted domain as the link text while the `href` points to a different, malicious destination. Users click without inspecting the URL.

**Mitigation (implemented):** All `http(s)` links in HTML email are given `target="_blank" rel="noopener noreferrer"` so they open safely in a new tab. For links where the visible text looks like a URL (starts with `http(s)://` or `www.`) but the registrable domain of the text differs from the registrable domain of the `href`, the link is highlighted in red with a wavy underline and a `⚠` suffix, and a descriptive `title` is added. A warning banner is shown above the email body with the count of suspicious links detected. The detection and styling happen client-side in the HTML processing pipeline, before the HTML is set as `srcdoc`.

## 4. Session credential storage

**Threat:** IMAP/SMTP credentials stored server-side can be exfiltrated if the database is compromised.

**Mitigation (implemented):** Credentials are encrypted with AES-256-GCM. The encryption key is derived via HKDF-SHA256 from two independent secrets: the server-side `SESSION_SECRET` (never leaves the server) and a per-session 16-byte random nonce stored in the browser cookie. Neither alone is sufficient — recovering the plaintext password requires both the database row (for the ciphertext) and either the server secret or the individual session cookie.

## 5. Cookie / session hijacking

**Threat:** A stolen session cookie grants full account access, including the ability to read, send, and delete mail.

**Mitigation (implemented):**
- `letrvu_session`: `HttpOnly`, `SameSite=Strict`, and `Secure` (when `SECURE_COOKIES=true`) are set.
- `letrvu_csrf`: `SameSite=Strict` and `Secure` (when `SECURE_COOKIES=true`) are set. `HttpOnly` is intentionally omitted so JavaScript can read the token.
- Set `SECURE_COOKIES=true` in production whenever the app is served over HTTPS.

Note: User-Agent binding was considered and rejected. The UA is present in the same HTTP request as the cookie, so any realistic theft scenario (network sniffing, XSS, devtools) gives the attacker both simultaneously. It adds schema complexity and spontaneous logouts on browser updates for no meaningful security gain.

## 6. CSRF

**Threat:** A malicious third-party site triggers a state-changing API call (send message, delete message) on behalf of a logged-in user whose cookie is sent automatically by the browser.

**Mitigation (implemented):** `SameSite=Strict` on session cookies prevents the cookie from being sent on cross-site requests, which covers most CSRF scenarios. As belt-and-suspenders, a double-submit CSRF token is required on all mutating API endpoints: the server sets a non-HttpOnly `letrvu_csrf` cookie on login, and the frontend reads it and sends it as an `X-CSRF-Token` header. The server validates that both values match using constant-time comparison.

## 7. IMAP/SMTP TLS verification

**Threat:** The `IMAP_INSECURE_TLS=true` environment variable disables TLS certificate validation. On a real mail server this exposes credentials and message content to network interception.

**Mitigation (partial):** This flag must never be set in production. The UI should display a visible warning when the server connection was established without certificate verification.

## 8. Content-Security-Policy for the app shell

**Threat:** Even with email-body isolation, reflected or stored XSS in the Vue application itself (e.g., via subject lines, sender names, or folder names rendered without sanitization) can execute code.

**Mitigation (implemented):** The Go server sets the following headers on every response:

```
Content-Security-Policy: default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data:; font-src 'self'; object-src 'none'; frame-ancestors 'none'; connect-src 'self'
X-Content-Type-Options: nosniff
X-Frame-Options: DENY
```

The `srcdoc` iframe is governed by its own `sandbox` attribute, not the parent page's CSP, so this header does not conflict with HTML email rendering.

---

## Priority summary

| Priority | Item |
|---|---|
| ~~High~~ Done | ~~Change `sandbox="allow-same-origin"` to `sandbox` in `MessageView.vue`~~ |
| ~~Medium~~ Done | ~~HTML sanitization — DOMPurify in the frontend before setting `srcdoc`~~ |
| ~~Medium~~ Done | ~~Add `Content-Security-Policy` header in the Go HTTP server~~ |
| ~~Low~~ Done | ~~CSRF double-submit cookie protection on all mutating API endpoints~~ |
| ~~High~~ Done | ~~Audit session cookie flags (`HttpOnly`, `Secure`, `SameSite=Strict`)~~ |
| ~~Low~~ Done | ~~Link destination warning for mismatched href text and href URL~~ |
| Low | Per-sender "always show images" preference persisted in settings |
| Low | UI warning when `IMAP_INSECURE_TLS=true` is active |
