package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/jsuto/letrvu/internal/filters"
)

func (h *handler) listFilters(w http.ResponseWriter, r *http.Request) {
	sess := h.sessionFrom(r)
	list, err := h.filters.List(sess.Username, sess.IMAPHost)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResp(err.Error()))
		return
	}
	if list == nil {
		list = []filters.Filter{}
	}
	writeJSON(w, http.StatusOK, list)
}

func (h *handler) createFilter(w http.ResponseWriter, r *http.Request) {
	sess := h.sessionFrom(r)
	var f filters.Filter
	if err := json.NewDecoder(r.Body).Decode(&f); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResp("invalid request body"))
		return
	}
	if strings.TrimSpace(f.Name) == "" {
		writeJSON(w, http.StatusBadRequest, errorResp("name is required"))
		return
	}
	f.Enabled = true // default to enabled
	id, err := h.filters.Create(sess.Username, sess.IMAPHost, f)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResp(err.Error()))
		return
	}
	f.ID = id
	h.syncFiltersToSieve(sess.Username, sess.IMAPHost, sess.Password)
	writeJSON(w, http.StatusOK, f)
}

func (h *handler) updateFilter(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResp("invalid id"))
		return
	}
	sess := h.sessionFrom(r)
	var f filters.Filter
	if err := json.NewDecoder(r.Body).Decode(&f); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResp("invalid request body"))
		return
	}
	if strings.TrimSpace(f.Name) == "" {
		writeJSON(w, http.StatusBadRequest, errorResp("name is required"))
		return
	}
	if err := h.filters.Update(id, sess.Username, sess.IMAPHost, f); err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResp(err.Error()))
		return
	}
	h.syncFiltersToSieve(sess.Username, sess.IMAPHost, sess.Password)
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *handler) deleteFilter(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResp("invalid id"))
		return
	}
	sess := h.sessionFrom(r)
	if err := h.filters.Delete(id, sess.Username, sess.IMAPHost); err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResp(err.Error()))
		return
	}
	h.syncFiltersToSieve(sess.Username, sess.IMAPHost, sess.Password)
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *handler) reorderFilters(w http.ResponseWriter, r *http.Request) {
	sess := h.sessionFrom(r)
	var body struct {
		IDs []int64 `json:"ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || len(body.IDs) == 0 {
		writeJSON(w, http.StatusBadRequest, errorResp("ids required"))
		return
	}
	if err := h.filters.Reorder(body.IDs, sess.Username, sess.IMAPHost); err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResp(err.Error()))
		return
	}
	h.syncFiltersToSieve(sess.Username, sess.IMAPHost, sess.Password)
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// syncFiltersToSieve pushes the current filter (and vacation) rules to
// ManageSieve. It is called after every filter mutation.
func (h *handler) syncFiltersToSieve(username, imapHost, password string) {
	active, _ := h.rebuildAndUploadSieve(username, imapHost, password)
	activeStr := "false"
	if active {
		activeStr = "true"
	}
	_ = h.settings.Set(username, imapHost, map[string]string{
		"vacation_sieve_active": activeStr,
	})
}
