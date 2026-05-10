/**
 * Thin fetch wrapper that automatically attaches the CSRF token header to
 * mutating requests (POST, PUT, PATCH, DELETE).
 *
 * The server sets a non-HttpOnly `letrvu_csrf` cookie on login. This function
 * reads that cookie and sends its value as `X-CSRF-Token`, implementing the
 * double-submit cookie pattern.
 */

const SAFE_METHODS = new Set(['GET', 'HEAD', 'OPTIONS'])

function getCSRFToken() {
  const match = document.cookie.match(/(?:^|;\s*)letrvu_csrf=([^;]+)/)
  return match ? match[1] : ''
}

export function apiFetch(url, options = {}) {
  const method = (options.method ?? 'GET').toUpperCase()
  const headers = new Headers(options.headers)

  if (!SAFE_METHODS.has(method)) {
    headers.set('X-CSRF-Token', getCSRFToken())
  }

  return fetch(url, { ...options, headers })
}
