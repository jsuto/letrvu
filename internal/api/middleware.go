package api

import (
	"crypto/subtle"
	"net/http"
)

const csp = "default-src 'self'; " +
	"script-src 'self'; " +
	"style-src 'self' 'unsafe-inline'; " +
	"img-src 'self' data: https: http:; " +
	"font-src 'self'; " +
	"object-src 'none'; " +
	"frame-ancestors 'none'; " +
	"connect-src 'self'"

// securityHeaders adds security-related HTTP response headers to every reply.
func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Security-Policy", csp)
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		next.ServeHTTP(w, r)
	})
}

// checkCSRF validates the double-submit CSRF token for mutating requests.
// Safe methods (GET, HEAD, OPTIONS) are skipped.
// The letrvu_csrf cookie value must match the X-CSRF-Token request header.
func checkCSRF(w http.ResponseWriter, r *http.Request) bool {
	switch r.Method {
	case http.MethodGet, http.MethodHead, http.MethodOptions:
		return true
	}
	cookie, err := r.Cookie("letrvu_csrf")
	if err != nil || cookie.Value == "" {
		writeJSON(w, http.StatusForbidden, errorResp("csrf token missing"))
		return false
	}
	header := r.Header.Get("X-CSRF-Token")
	if subtle.ConstantTimeCompare([]byte(header), []byte(cookie.Value)) != 1 {
		writeJSON(w, http.StatusForbidden, errorResp("csrf token invalid"))
		return false
	}
	return true
}
