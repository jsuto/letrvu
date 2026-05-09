package api

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"

	"github.com/yourusername/letrvu/internal/imap"
	"github.com/yourusername/letrvu/internal/session"
)

type handler struct {
	sessions *session.Store
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

	sess, err := h.sessions.Create(body.IMAPHost, body.IMAPPort, body.SMTPHost, body.SMTPPort, body.Username, body.Password)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResp("could not create session"))
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "letrvu_session",
		Value:    sess.ID,
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

	sess := h.sessionFrom(r)
	c, err := imapConnect(sess)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, errorResp("imap connection failed"))
		return
	}
	defer c.Close()

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
	uidStr := r.PathValue("uid")
	uidVal, err := strconv.ParseUint(uidStr, 10, 32)
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
	writeJSON(w, http.StatusNotImplemented, errorResp("not implemented"))
}

func (h *handler) markRead(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusNotImplemented, errorResp("not implemented"))
}

func (h *handler) sendMessage(w http.ResponseWriter, r *http.Request) {
	// TODO: decode body, use smtp.Send()
	writeJSON(w, http.StatusNotImplemented, errorResp("not implemented"))
}

// events streams server-sent events for real-time new mail notifications.
// Relies on IMAP IDLE in the imap package (to be implemented).
func (h *handler) events(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	// TODO: subscribe to IMAP IDLE notifications and stream events
	<-r.Context().Done()
}
