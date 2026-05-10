package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/jsuto/letrvu/internal/calendar"
	"github.com/jsuto/letrvu/internal/contacts"
	"github.com/jsuto/letrvu/internal/imap"
	"github.com/jsuto/letrvu/internal/session"
	"github.com/jsuto/letrvu/internal/settings"
	"github.com/jsuto/letrvu/internal/smtp"
)

// ServerConfig holds server-level IMAP/SMTP defaults exposed via /api/config
// to pre-fill the login form.
type ServerConfig struct {
	IMAPHost string `json:"imap_host"`
	IMAPPort int    `json:"imap_port"`
	SMTPHost string `json:"smtp_host"`
	SMTPPort int    `json:"smtp_port"`
}

type handler struct {
	sessions *session.Store
	settings *settings.Store
	contacts *contacts.Store
	calendar *calendar.Store
	config   ServerConfig
}

// requireAuth is middleware that checks for a valid session cookie.
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
		writeJSON(w, http.StatusUnauthorized, errorResp("imap authentication failed"))
		return
	}
	c.Close()

	cookieVal, err := h.sessions.Create(body.IMAPHost, body.IMAPPort, body.SMTPHost, body.SMTPPort, body.Username, body.Password)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResp("could not create session"))
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "letrvu_session",
		Value:    cookieVal,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	})
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *handler) logout(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("letrvu_session")
	if err == nil {
		h.sessions.Delete(cookie.Value)
	}
	http.SetCookie(w, &http.Cookie{Name: "letrvu_session", MaxAge: -1, Path: "/"})
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *handler) listFolders(w http.ResponseWriter, r *http.Request) {
	sess := h.sessionFrom(r)
	c, err := imapConnect(sess)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, errorResp("imap connection failed"))
		return
	}
	defer c.Close()

	folders, err := c.ListFolders()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResp(err.Error()))
		return
	}
	writeJSON(w, http.StatusOK, folders)
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
		To      []string `json:"to"`
		CC      []string `json:"cc"`
		Subject string   `json:"subject"`
		Text    string   `json:"text"`
		HTML    string   `json:"html"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResp("invalid request body"))
		return
	}

	sess := h.sessionFrom(r)
	if err := smtp.Send(smtp.Config{
		Host:     sess.SMTPHost,
		Port:     sess.SMTPPort,
		Username: sess.Username,
		Password: sess.Password,
	}, smtp.Message{
		From:    sess.Username,
		To:      body.To,
		CC:      body.CC,
		Subject: body.Subject,
		Text:    body.Text,
		HTML:    body.HTML,
	}); err != nil {
		writeJSON(w, http.StatusBadGateway, errorResp(err.Error()))
		return
	}
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
	writeJSON(w, http.StatusOK, s)
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
			data, _ := json.Marshal(ev)
			fmt.Fprintf(w, "event: mailbox\ndata: %s\n\n", data)
			flusher.Flush()
		}
	}
}
