package api

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/jsuto/letrvu/internal/calendar"
	"github.com/jsuto/letrvu/internal/contacts"
	"github.com/jsuto/letrvu/internal/imap"
	"github.com/jsuto/letrvu/internal/session"
	"github.com/jsuto/letrvu/internal/settings"
	"github.com/jsuto/letrvu/internal/smtp"
)

func randomHex(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// ServerConfig holds server-level IMAP/SMTP defaults exposed via /api/config
// to pre-fill the login form.
type ServerConfig struct {
	IMAPHost        string        `json:"imap_host"`
	IMAPPort        int           `json:"imap_port"`
	SMTPHost        string        `json:"smtp_host"`
	SMTPPort        int           `json:"smtp_port"`
	SecureCookies   bool          `json:"-"`
	FolderCacheTTL  time.Duration `json:"-"`
	// TrustedProxy is the IP or CIDR of the reverse proxy. When non-nil,
	// X-Forwarded-For / X-Real-IP are trusted only if the TCP connection comes
	// from within that range. Nil disables proxy header reading entirely.
	TrustedProxy    *net.IPNet    `json:"-"`
	// InternalDomains is the set of domains that belong to this organisation.
	// Messages whose From domain is not in this list are flagged as external
	// in the UI — but only when Authentication-Results headers are present so
	// we are not making judgements based on spoofable data.
	InternalDomains []string `json:"-"`
}


type handler struct {
	sessions     *session.Store
	settings     *settings.Store
	contacts     *contacts.Store
	calendar     *calendar.Store
	config       ServerConfig
	folderCache  *folderCache
}

// clientIP returns the real client IP address. If TrustedProxy is configured
// and the TCP connection originates from within that CIDR, X-Forwarded-For
// (leftmost entry) or X-Real-IP is used. The CIDR check prevents a client
// that bypasses the proxy from forging its own IP in the audit log.
func (h *handler) clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	if h.config.TrustedProxy != nil {
		if ip := net.ParseIP(host); ip != nil && h.config.TrustedProxy.Contains(ip) {
			if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
				// X-Forwarded-For may be "client, proxy1, proxy2" — leftmost is original client.
				if i := strings.IndexByte(xff, ','); i > 0 {
					return strings.TrimSpace(xff[:i])
				}
				return strings.TrimSpace(xff)
			}
			if xri := r.Header.Get("X-Real-IP"); xri != "" {
				return strings.TrimSpace(xri)
			}
		}
	}
	return host
}

// requireAuth is middleware that checks for a valid session cookie and, for
// mutating requests, validates the CSRF token.
func (h *handler) requireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("letrvu_session")
		if err != nil {
			writeJSON(w, http.StatusUnauthorized, errorResp("unauthorized"))
			return
		}
		if _, ok := h.sessions.Get(cookie.Value); !ok {
			writeJSON(w, http.StatusUnauthorized, errorResp("session expired"))
			return
		}
		if !checkCSRF(w, r) {
			return
		}
		next(w, r)
	}
}

// sessionFrom extracts the session for an authenticated request.
func (h *handler) sessionFrom(r *http.Request) *session.Session {
	cookie, _ := r.Cookie("letrvu_session")
	sess, _ := h.sessions.Get(cookie.Value)
	return sess
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

type errorResp string

func (e errorResp) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]string{"error": string(e)})
}

// imapConnect opens an authenticated IMAP connection using session credentials.
func imapConnect(sess *session.Session) (*imap.Client, error) {
	return imap.Connect(sess.IMAPHost, sess.IMAPPort, sess.Username, sess.Password)
}

// login authenticates the user and sets a session cookie.
// It dials IMAP to verify credentials before creating a session.
func (h *handler) login(w http.ResponseWriter, r *http.Request) {
	var body struct {
		IMAPHost string `json:"imap_host"`
		IMAPPort int    `json:"imap_port"`
		SMTPHost string `json:"smtp_host"`
		SMTPPort int    `json:"smtp_port"`
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResp("invalid request body"))
		return
	}
	if body.IMAPPort == 0 {
		body.IMAPPort = 993
	}
	if body.SMTPPort == 0 {
		body.SMTPPort = 587
	}

	// Verify credentials by dialing IMAP before creating session.
	c, err := imap.Connect(body.IMAPHost, body.IMAPPort, body.Username, body.Password)
	if err != nil {
		log.Printf("audit: login_failed user=%s ip=%s imap=%s", body.Username, h.clientIP(r), body.IMAPHost)
		writeJSON(w, http.StatusUnauthorized, errorResp("imap authentication failed"))
		return
	}
	c.Close()

	cookieVal, err := h.sessions.Create(body.IMAPHost, body.IMAPPort, body.SMTPHost, body.SMTPPort, body.Username, body.Password)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResp("could not create session"))
		return
	}

	csrfToken, err := randomHex(32)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResp("could not create session"))
		return
	}

	log.Printf("audit: login user=%s ip=%s imap=%s", body.Username, h.clientIP(r), body.IMAPHost)

	http.SetCookie(w, &http.Cookie{
		Name:     "letrvu_session",
		Value:    cookieVal,
		Path:     "/",
		HttpOnly: true,
		Secure:   h.config.SecureCookies,
		SameSite: http.SameSiteStrictMode,
	})
	http.SetCookie(w, &http.Cookie{
		Name:     "letrvu_csrf",
		Value:    csrfToken,
		Path:     "/",
		HttpOnly: false, // must be readable by JS
		Secure:   h.config.SecureCookies,
		SameSite: http.SameSiteStrictMode,
	})
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *handler) logout(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("letrvu_session")
	if err == nil {
		if sess, ok := h.sessions.Get(cookie.Value); ok {
			log.Printf("audit: logout user=%s ip=%s", sess.Username, h.clientIP(r))
		}
		h.sessions.Delete(cookie.Value)
	}
	http.SetCookie(w, &http.Cookie{Name: "letrvu_session", MaxAge: -1, Path: "/"})
	http.SetCookie(w, &http.Cookie{Name: "letrvu_csrf", MaxAge: -1, Path: "/"})
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *handler) listFolders(w http.ResponseWriter, r *http.Request) {
	sess := h.sessionFrom(r)
	key := cacheKey(sess.Username, sess.IMAPHost)

	// Serve cached data immediately if available.
	if cached, ok := h.folderCache.get(key); ok {
		writeJSON(w, http.StatusOK, cached)
		// Refresh in the background if the entry is stale.
		if h.folderCache.stale(key) && h.folderCache.markRefreshing(key) {
			go h.refreshFolderCache(sess.IMAPHost, sess.IMAPPort, sess.Username, sess.Password, key)
		}
		return
	}

	// No cache yet — fetch synchronously so the first load has real data.
	folders, err := h.fetchAndCacheFolders(sess.IMAPHost, sess.IMAPPort, sess.Username, sess.Password, key)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, errorResp("imap connection failed"))
		return
	}
	writeJSON(w, http.StatusOK, folders)
}

func (h *handler) fetchAndCacheFolders(imapHost string, imapPort int, username, password, key string) ([]imap.Folder, error) {
	c, err := imapConnect(&session.Session{
		IMAPHost: imapHost, IMAPPort: imapPort,
		Username: username, Password: password,
	})
	if err != nil {
		return nil, err
	}
	defer c.Close()
	folders, err := c.ListFolders()
	if err != nil {
		return nil, err
	}
	h.folderCache.set(key, folders)
	return folders, nil
}

func (h *handler) refreshFolderCache(imapHost string, imapPort int, username, password, key string) {
	h.fetchAndCacheFolders(imapHost, imapPort, username, password, key) //nolint:errcheck
}

func (h *handler) listMessages(w http.ResponseWriter, r *http.Request) {
	folder, err := url.PathUnescape(r.PathValue("folder"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResp("invalid folder name"))
		return
	}

	sess := h.sessionFrom(r)
	c, err := imapConnect(sess)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, errorResp("imap connection failed"))
		return
	}
	defer c.Close()

	// Search mode when ?q= is provided.
	if q := r.URL.Query().Get("q"); q != "" {
		msgs, err := c.SearchMessages(folder, q)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, errorResp(err.Error()))
			return
		}
		writeJSON(w, http.StatusOK, msgs)
		return
	}

	page := 1
	pageSize := 50
	if p := r.URL.Query().Get("page"); p != "" {
		if n, err := strconv.Atoi(p); err == nil && n > 0 {
			page = n
		}
	}
	if ps := r.URL.Query().Get("page_size"); ps != "" {
		if n, err := strconv.Atoi(ps); err == nil && n > 0 && n <= 200 {
			pageSize = n
		}
	}

	msgs, err := c.ListMessages(folder, page, pageSize)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResp(err.Error()))
		return
	}
	writeJSON(w, http.StatusOK, msgs)
}

func (h *handler) getMessage(w http.ResponseWriter, r *http.Request) {
	folder, err := url.PathUnescape(r.PathValue("folder"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResp("invalid folder name"))
		return
	}
	uidVal, err := strconv.ParseUint(r.PathValue("uid"), 10, 32)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResp("invalid uid"))
		return
	}

	sess := h.sessionFrom(r)
	c, err := imapConnect(sess)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, errorResp("imap connection failed"))
		return
	}
	defer c.Close()

	msg, err := c.GetMessage(folder, uint32(uidVal))
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResp(err.Error()))
		return
	}
	writeJSON(w, http.StatusOK, msg)
}

func (h *handler) getMessageSource(w http.ResponseWriter, r *http.Request) {
	folder, err := url.PathUnescape(r.PathValue("folder"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResp("invalid folder name"))
		return
	}
	uidVal, err := strconv.ParseUint(r.PathValue("uid"), 10, 32)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResp("invalid uid"))
		return
	}

	sess := h.sessionFrom(r)
	c, err := imapConnect(sess)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, errorResp("imap connection failed"))
		return
	}
	defer c.Close()

	raw, err := c.GetRawMessage(folder, uint32(uidVal))
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResp(err.Error()))
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write(raw) //nolint:errcheck
}

func (h *handler) deleteMessage(w http.ResponseWriter, r *http.Request) {
	folder, err := url.PathUnescape(r.PathValue("folder"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResp("invalid folder name"))
		return
	}
	uidVal, err := strconv.ParseUint(r.PathValue("uid"), 10, 32)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResp("invalid uid"))
		return
	}

	sess := h.sessionFrom(r)
	c, err := imapConnect(sess)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, errorResp("imap connection failed"))
		return
	}
	defer c.Close()

	if err := c.DeleteMessage(folder, uint32(uidVal)); err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResp(err.Error()))
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *handler) markRead(w http.ResponseWriter, r *http.Request) {
	folder, err := url.PathUnescape(r.PathValue("folder"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResp("invalid folder name"))
		return
	}
	uidVal, err := strconv.ParseUint(r.PathValue("uid"), 10, 32)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResp("invalid uid"))
		return
	}

	var body struct {
		Read bool `json:"read"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResp("invalid request body"))
		return
	}

	sess := h.sessionFrom(r)
	c, err := imapConnect(sess)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, errorResp("imap connection failed"))
		return
	}
	defer c.Close()

	if err := c.MarkRead(folder, uint32(uidVal), body.Read); err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResp(err.Error()))
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *handler) markFlagged(w http.ResponseWriter, r *http.Request) {
	folder, err := url.PathUnescape(r.PathValue("folder"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResp("invalid folder name"))
		return
	}
	uidVal, err := strconv.ParseUint(r.PathValue("uid"), 10, 32)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResp("invalid uid"))
		return
	}
	var body struct {
		Flagged bool `json:"flagged"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResp("invalid request body"))
		return
	}
	sess := h.sessionFrom(r)
	c, err := imapConnect(sess)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, errorResp("imap connection failed"))
		return
	}
	defer c.Close()
	if err := c.MarkFlagged(folder, uint32(uidVal), body.Flagged); err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResp(err.Error()))
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *handler) moveMessage(w http.ResponseWriter, r *http.Request) {
	folder, err := url.PathUnescape(r.PathValue("folder"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResp("invalid folder name"))
		return
	}
	uidVal, err := strconv.ParseUint(r.PathValue("uid"), 10, 32)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResp("invalid uid"))
		return
	}
	var body struct {
		Dest string `json:"dest"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Dest == "" {
		writeJSON(w, http.StatusBadRequest, errorResp("dest folder required"))
		return
	}

	sess := h.sessionFrom(r)
	c, err := imapConnect(sess)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, errorResp("imap connection failed"))
		return
	}
	defer c.Close()

	if err := c.MoveMessage(folder, uint32(uidVal), body.Dest); err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResp(err.Error()))
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *handler) moveMessages(w http.ResponseWriter, r *http.Request) {
	folder, err := url.PathUnescape(r.PathValue("folder"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResp("invalid folder name"))
		return
	}
	var body struct {
		UIDs []uint32 `json:"uids"`
		Dest string   `json:"dest"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || len(body.UIDs) == 0 || body.Dest == "" {
		writeJSON(w, http.StatusBadRequest, errorResp("uids and dest required"))
		return
	}

	sess := h.sessionFrom(r)
	c, err := imapConnect(sess)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, errorResp("imap connection failed"))
		return
	}
	defer c.Close()

	if err := c.MoveMessages(folder, body.UIDs, body.Dest); err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResp(err.Error()))
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *handler) sendMessage(w http.ResponseWriter, r *http.Request) {
	var body struct {
		FromName  string   `json:"from_name"`
		FromEmail string   `json:"from_email"`
		To        []string `json:"to"`
		CC        []string `json:"cc"`
		Subject   string   `json:"subject"`
		Text      string   `json:"text"`
		HTML      string   `json:"html"`
		Attachments []struct {
			Filename    string `json:"filename"`
			ContentType string `json:"content_type"`
			Data        string `json:"data"` // base64-encoded
		} `json:"attachments,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResp("invalid request body"))
		return
	}

	sess := h.sessionFrom(r)

	// Build the RFC 5322 From: header from the selected identity.
	// Fall back to the authenticated username if no identity was chosen.
	fromEmail := body.FromEmail
	if fromEmail == "" {
		fromEmail = sess.Username
	}
	var fromHeader string
	if body.FromName != "" {
		fromHeader = fmt.Sprintf("%s <%s>", body.FromName, fromEmail)
	} else {
		fromHeader = fromEmail
	}

	var attachments []smtp.Attachment
	for _, a := range body.Attachments {
		data, err := base64.StdEncoding.DecodeString(a.Data)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, errorResp("invalid attachment data"))
			return
		}
		attachments = append(attachments, smtp.Attachment{
			Filename:    a.Filename,
			ContentType: a.ContentType,
			Data:        data,
		})
	}

	smtpMsg := smtp.Message{
		From:         fromHeader,
		EnvelopeFrom: sess.Username, // authenticated address for bounce routing
		To:           body.To,
		CC:           body.CC,
		Subject:      body.Subject,
		Text:         body.Text,
		HTML:         body.HTML,
		Attachments:  attachments,
	}

	if err := smtp.Send(smtp.Config{
		Host:     sess.SMTPHost,
		Port:     sess.SMTPPort,
		Username: sess.Username,
		Password: sess.Password,
	}, smtpMsg); err != nil {
		log.Printf("audit: send_failed user=%s ip=%s to=%d cc=%d attachments=%d err=%v",
			sess.Username, h.clientIP(r), len(body.To), len(body.CC), len(attachments), err)
		writeJSON(w, http.StatusBadGateway, errorResp(err.Error()))
		return
	}
	log.Printf("audit: send user=%s ip=%s to=%d cc=%d attachments=%d",
		sess.Username, h.clientIP(r), len(body.To), len(body.CC), len(attachments))

	// Best-effort: save a copy to the Sent folder via IMAP APPEND.
	// A failure here is logged but does not affect the send response.
	go func() {
		raw := smtp.BuildRFC822(smtpMsg)
		c, err := imapConnect(sess)
		if err != nil {
			log.Printf("warn: sent_copy_failed user=%s: imap connect: %v", sess.Username, err)
			return
		}
		defer c.Close()
		if err := c.SaveSent(raw); err != nil {
			log.Printf("warn: sent_copy_failed user=%s: %v", sess.Username, err)
		}
	}()

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *handler) subscribeFolder(w http.ResponseWriter, r *http.Request) {
	folder, err := url.PathUnescape(r.PathValue("folder"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResp("invalid folder name"))
		return
	}
	sess := h.sessionFrom(r)
	c, err := imapConnect(sess)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, errorResp("imap connection failed"))
		return
	}
	defer c.Close()
	if err := c.Subscribe(folder); err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResp(err.Error()))
		return
	}
	h.folderCache.invalidate(cacheKey(sess.Username, sess.IMAPHost))
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *handler) unsubscribeFolder(w http.ResponseWriter, r *http.Request) {
	folder, err := url.PathUnescape(r.PathValue("folder"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResp("invalid folder name"))
		return
	}
	sess := h.sessionFrom(r)
	c, err := imapConnect(sess)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, errorResp("imap connection failed"))
		return
	}
	defer c.Close()
	if err := c.Unsubscribe(folder); err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResp(err.Error()))
		return
	}
	h.folderCache.invalidate(cacheKey(sess.Username, sess.IMAPHost))
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *handler) saveDraft(w http.ResponseWriter, r *http.Request) {
	var body struct {
		FromName  string   `json:"from_name"`
		FromEmail string   `json:"from_email"`
		To        []string `json:"to"`
		CC        []string `json:"cc"`
		Subject   string   `json:"subject"`
		Text      string   `json:"text"`
		HTML      string   `json:"html"`
		Attachments []struct {
			Filename    string `json:"filename"`
			ContentType string `json:"content_type"`
			Data        string `json:"data"` // base64-encoded
		} `json:"attachments,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResp("invalid request body"))
		return
	}

	sess := h.sessionFrom(r)

	fromEmail := body.FromEmail
	if fromEmail == "" {
		fromEmail = sess.Username
	}
	var fromHeader string
	if body.FromName != "" {
		fromHeader = fmt.Sprintf("%s <%s>", body.FromName, fromEmail)
	} else {
		fromHeader = fromEmail
	}

	var attachments []smtp.Attachment
	for _, a := range body.Attachments {
		data, err := base64.StdEncoding.DecodeString(a.Data)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, errorResp("invalid attachment data"))
			return
		}
		attachments = append(attachments, smtp.Attachment{
			Filename:    a.Filename,
			ContentType: a.ContentType,
			Data:        data,
		})
	}

	raw := smtp.BuildRFC822(smtp.Message{
		From:        fromHeader,
		To:          body.To,
		CC:          body.CC,
		Subject:     body.Subject,
		Text:        body.Text,
		HTML:        body.HTML,
		Attachments: attachments,
	})

	c, err := imapConnect(sess)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, errorResp("imap connection failed"))
		return
	}
	defer c.Close()

	if err := c.SaveDraft(raw); err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResp(err.Error()))
		return
	}

	log.Printf("audit: draft_saved user=%s ip=%s", sess.Username, h.clientIP(r))
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *handler) downloadAttachment(w http.ResponseWriter, r *http.Request) {
	folder, err := url.PathUnescape(r.PathValue("folder"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResp("invalid folder name"))
		return
	}
	uidVal, err := strconv.ParseUint(r.PathValue("uid"), 10, 32)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResp("invalid uid"))
		return
	}
	index, err := strconv.Atoi(r.PathValue("index"))
	if err != nil || index < 0 {
		writeJSON(w, http.StatusBadRequest, errorResp("invalid attachment index"))
		return
	}

	sess := h.sessionFrom(r)
	c, err := imapConnect(sess)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, errorResp("imap connection failed"))
		return
	}
	defer c.Close()

	att, data, err := c.DownloadAttachment(folder, uint32(uidVal), index)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResp(err.Error()))
		return
	}

	ct := att.ContentType
	if ct == "" {
		ct = "application/octet-stream"
	}
	w.Header().Set("Content-Type", ct)
	w.Header().Set("Content-Length", strconv.Itoa(len(data)))
	if att.Filename != "" {
		w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename=%q`, att.Filename))
	}
	w.Write(data) //nolint:errcheck
}

// getConfig returns server-configured IMAP/SMTP defaults for the login form.
// This endpoint is public (no auth required).
func (h *handler) getConfig(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, h.config)
}

func (h *handler) getSettings(w http.ResponseWriter, r *http.Request) {
	sess := h.sessionFrom(r)
	s, err := h.settings.Get(sess.Username, sess.IMAPHost)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResp(err.Error()))
		return
	}
	// Build response as map[string]any so we can mix string user-settings with
	// server-injected values of other types (e.g. the domains slice).
	out := make(map[string]any, len(s)+2)
	for k, v := range s {
		out[k] = v
	}
	// Inject the authenticated username so the client can show it as the
	// default From address when no identities are configured.
	out["username"] = sess.Username
	// Inject the server-configured internal domains so the client can flag
	// messages from outside the organisation.
	out["internal_domains"] = h.config.InternalDomains
	writeJSON(w, http.StatusOK, out)
}

func (h *handler) updateSettings(w http.ResponseWriter, r *http.Request) {
	sess := h.sessionFrom(r)
	var values map[string]string
	if err := json.NewDecoder(r.Body).Decode(&values); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResp("invalid request body"))
		return
	}
	if err := h.settings.Set(sess.Username, sess.IMAPHost, values); err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResp(err.Error()))
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// events streams server-sent events for real-time new mail notifications via
// IMAP IDLE. The optional ?folder= query param selects the watched folder
// (defaults to INBOX).
func (h *handler) events(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeJSON(w, http.StatusInternalServerError, errorResp("streaming not supported"))
		return
	}

	folder := r.URL.Query().Get("folder")
	if folder == "" {
		folder = "INBOX"
	}

	sess := h.sessionFrom(r)
	mailEvents, cancel, err := imap.WatchFolder(
		sess.IMAPHost, sess.IMAPPort, sess.Username, sess.Password, folder,
	)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, errorResp(err.Error()))
		return
	}
	defer cancel()

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	// Initial comment so the browser knows the stream is open.
	fmt.Fprintf(w, ": connected\n\n")
	flusher.Flush()

	for {
		select {
		case <-r.Context().Done():
			return
		case ev, ok := <-mailEvents:
			if !ok {
				return // IDLE connection dropped
			}
			// Invalidate folder cache so unseen counts refresh on next fetch.
			h.folderCache.invalidate(cacheKey(sess.Username, sess.IMAPHost))
			data, _ := json.Marshal(ev)
			fmt.Fprintf(w, "event: mailbox\ndata: %s\n\n", data)
			flusher.Flush()
		}
	}
}
