package api

import (
	"encoding/base64"
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"github.com/jsuto/letrvu/internal/totp"
)

// totpVerify completes a pending login by verifying a TOTP code or recovery code.
// This is a public endpoint (no session yet) — CSRF is not checked, protected
// instead by the short-lived letrvu_totp_pending HttpOnly SameSite=Strict cookie.
func (h *handler) totpVerify(w http.ResponseWriter, r *http.Request) {
	ip := h.clientIP(r)

	if blocked, remaining := h.loginLimiter.blocked(ip); blocked {
		retryAfter := int(remaining.Seconds()) + 1
		w.Header().Set("Retry-After", strconv.Itoa(retryAfter))
		writeJSON(w, http.StatusTooManyRequests, errorResp("too many attempts, try again later"))
		return
	}

	pendingCookie, err := r.Cookie("letrvu_totp_pending")
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, errorResp("no pending login"))
		return
	}

	pending, ok := h.pending.Get(pendingCookie.Value)
	if !ok {
		clearPendingCookie(w, h.config.SecureCookies)
		writeJSON(w, http.StatusUnauthorized, errorResp("pending login expired"))
		return
	}

	var body struct {
		Code string `json:"code"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Code == "" {
		writeJSON(w, http.StatusBadRequest, errorResp("code is required"))
		return
	}

	secret, _, hasSecret := h.totp.GetSecret(pending.Username, pending.IMAPHost)
	if !hasSecret {
		clearPendingCookie(w, h.config.SecureCookies)
		writeJSON(w, http.StatusUnauthorized, errorResp("2FA not configured"))
		return
	}

	valid := totp.Validate(body.Code, secret) || h.totp.ConsumeRecoveryCode(pending.Username, pending.IMAPHost, body.Code)
	if !valid {
		h.loginLimiter.recordFailure(ip)
		log.Printf("audit: totp_verify_failed user=%s ip=%s", pending.Username, ip)
		writeJSON(w, http.StatusUnauthorized, errorResp("invalid code"))
		return
	}

	h.loginLimiter.recordSuccess(ip)
	h.pending.Delete(pendingCookie.Value)
	clearPendingCookie(w, h.config.SecureCookies)

	cookieVal, err := h.sessions.Create(
		pending.IMAPHost, pending.IMAPPort,
		pending.SMTPHost, pending.SMTPPort,
		pending.Username, pending.Password, pending.UserAgent,
	)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResp("could not create session"))
		return
	}

	csrfToken, err := randomHex(32)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResp("could not create session"))
		return
	}

	log.Printf("audit: totp_login user=%s ip=%s imap=%s", pending.Username, ip, pending.IMAPHost)

	http.SetCookie(w, &http.Cookie{
		Name: "letrvu_session", Value: cookieVal, Path: "/",
		HttpOnly: true, Secure: h.config.SecureCookies, SameSite: http.SameSiteStrictMode,
	})
	http.SetCookie(w, &http.Cookie{
		Name: "letrvu_csrf", Value: csrfToken, Path: "/",
		HttpOnly: false, Secure: h.config.SecureCookies, SameSite: http.SameSiteStrictMode,
	})
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// totp2faSetup generates a new TOTP secret for enrollment. The secret is stored
// as pending (not yet active) until the user verifies a code via totp2faEnable.
func (h *handler) totp2faSetup(w http.ResponseWriter, r *http.Request) {
	sess := h.sessionFrom(r)

	secret, otpauthURL, qrPNG, err := totp.Generate("letrvu", sess.Username)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResp("could not generate TOTP key"))
		return
	}

	if err := h.totp.SavePendingSecret(sess.Username, sess.IMAPHost, secret); err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResp("could not save TOTP secret"))
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"secret":       secret,
		"otpauth_url":  otpauthURL,
		"qr_png_b64":   base64.StdEncoding.EncodeToString(qrPNG),
	})
}

// totp2faEnable verifies the enrollment code and activates 2FA for the user.
// Also generates and returns 10 recovery codes (shown once).
func (h *handler) totp2faEnable(w http.ResponseWriter, r *http.Request) {
	sess := h.sessionFrom(r)

	var body struct {
		Code string `json:"code"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Code == "" {
		writeJSON(w, http.StatusBadRequest, errorResp("code is required"))
		return
	}

	secret, _, ok := h.totp.GetSecret(sess.Username, sess.IMAPHost)
	if !ok {
		writeJSON(w, http.StatusBadRequest, errorResp("no pending TOTP setup — call GET /api/2fa/setup first"))
		return
	}

	if !totp.Validate(body.Code, secret) {
		writeJSON(w, http.StatusUnauthorized, errorResp("invalid code"))
		return
	}

	if err := h.totp.Enable(sess.Username, sess.IMAPHost); err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResp("could not enable 2FA"))
		return
	}

	plain, hashes, err := totp.GenerateRecoveryCodes()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResp("could not generate recovery codes"))
		return
	}
	if err := h.totp.SaveRecoveryCodes(sess.Username, sess.IMAPHost, hashes); err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResp("could not save recovery codes"))
		return
	}

	log.Printf("audit: totp_enabled user=%s ip=%s", sess.Username, h.clientIP(r))
	writeJSON(w, http.StatusOK, map[string]any{"recovery_codes": plain})
}

// totp2faDisable disables 2FA after the user confirms with a current TOTP or recovery code.
func (h *handler) totp2faDisable(w http.ResponseWriter, r *http.Request) {
	sess := h.sessionFrom(r)

	var body struct {
		Code string `json:"code"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Code == "" {
		writeJSON(w, http.StatusBadRequest, errorResp("code is required"))
		return
	}

	secret, enabled, ok := h.totp.GetSecret(sess.Username, sess.IMAPHost)
	if !ok || !enabled {
		writeJSON(w, http.StatusBadRequest, errorResp("2FA is not enabled"))
		return
	}

	valid := totp.Validate(body.Code, secret) || h.totp.ConsumeRecoveryCode(sess.Username, sess.IMAPHost, body.Code)
	if !valid {
		writeJSON(w, http.StatusUnauthorized, errorResp("invalid code"))
		return
	}

	if err := h.totp.Delete(sess.Username, sess.IMAPHost); err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResp("could not disable 2FA"))
		return
	}

	log.Printf("audit: totp_disabled user=%s ip=%s", sess.Username, h.clientIP(r))
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// totp2faRecoveryCodes regenerates recovery codes. Requires a valid TOTP code to confirm.
func (h *handler) totp2faRecoveryCodes(w http.ResponseWriter, r *http.Request) {
	sess := h.sessionFrom(r)

	var body struct {
		Code string `json:"code"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Code == "" {
		writeJSON(w, http.StatusBadRequest, errorResp("code is required"))
		return
	}

	secret, enabled, ok := h.totp.GetSecret(sess.Username, sess.IMAPHost)
	if !ok || !enabled {
		writeJSON(w, http.StatusBadRequest, errorResp("2FA is not enabled"))
		return
	}

	if !totp.Validate(body.Code, secret) {
		writeJSON(w, http.StatusUnauthorized, errorResp("invalid TOTP code"))
		return
	}

	plain, hashes, err := totp.GenerateRecoveryCodes()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResp("could not generate recovery codes"))
		return
	}
	if err := h.totp.SaveRecoveryCodes(sess.Username, sess.IMAPHost, hashes); err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResp("could not save recovery codes"))
		return
	}

	log.Printf("audit: totp_recovery_codes_regenerated user=%s ip=%s", sess.Username, h.clientIP(r))
	writeJSON(w, http.StatusOK, map[string]any{"recovery_codes": plain})
}

func clearPendingCookie(w http.ResponseWriter, secure bool) {
	http.SetCookie(w, &http.Cookie{
		Name: "letrvu_totp_pending", MaxAge: -1, Path: "/",
		HttpOnly: true, Secure: secure, SameSite: http.SameSiteStrictMode,
	})
}
