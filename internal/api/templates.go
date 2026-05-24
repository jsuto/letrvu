package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/jsuto/letrvu/internal/templates"
)

func (h *handler) listTemplates(w http.ResponseWriter, r *http.Request) {
	sess := h.sessionFrom(r)
	list, err := h.templates.List(sess.Username, sess.IMAPHost)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResp(err.Error()))
		return
	}
	if list == nil {
		list = []templates.Template{}
	}
	writeJSON(w, http.StatusOK, list)
}

func (h *handler) createTemplate(w http.ResponseWriter, r *http.Request) {
	sess := h.sessionFrom(r)
	var t templates.Template
	if err := json.NewDecoder(r.Body).Decode(&t); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResp("invalid request body"))
		return
	}
	if strings.TrimSpace(t.Name) == "" {
		writeJSON(w, http.StatusBadRequest, errorResp("name is required"))
		return
	}
	id, err := h.templates.Create(sess.Username, sess.IMAPHost, t)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResp(err.Error()))
		return
	}
	t.ID = id
	writeJSON(w, http.StatusOK, t)
}

func (h *handler) updateTemplate(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResp("invalid id"))
		return
	}
	sess := h.sessionFrom(r)
	var t templates.Template
	if err := json.NewDecoder(r.Body).Decode(&t); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResp("invalid request body"))
		return
	}
	if strings.TrimSpace(t.Name) == "" {
		writeJSON(w, http.StatusBadRequest, errorResp("name is required"))
		return
	}
	if err := h.templates.Update(id, sess.Username, sess.IMAPHost, t); err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResp(err.Error()))
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *handler) deleteTemplate(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResp("invalid id"))
		return
	}
	sess := h.sessionFrom(r)
	if err := h.templates.Delete(id, sess.Username, sess.IMAPHost); err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResp(err.Error()))
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
