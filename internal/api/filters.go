package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/jsuto/letrvu/internal/filters"
	"github.com/jsuto/letrvu/internal/imap"
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

// applyFilters manually runs the Go-based filter engine against messages in a
// folder. This is the fallback path when Sieve is not available, or for
// retroactively applying new rules to existing mail.
func (h *handler) applyFilters(w http.ResponseWriter, r *http.Request) {
	sess := h.sessionFrom(r)

	var body struct {
		Folder string `json:"folder"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Folder == "" {
		writeJSON(w, http.StatusBadRequest, errorResp("folder is required"))
		return
	}

	filterList, err := h.filters.List(sess.Username, sess.IMAPHost)
	if err != nil || len(filterList) == 0 {
		writeJSON(w, http.StatusOK, map[string]any{"applied": 0})
		return
	}

	c, err := imapConnect(sess)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, errorResp("imap connection failed"))
		return
	}
	defer c.Close()

	msgs, err := c.ListMessages(body.Folder, 1, 500)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResp(err.Error()))
		return
	}

	applied := 0
	for _, msg := range msgs {
		for _, f := range filterList {
			in := filters.MatchInput{
				Subject:       msg.Subject,
				From:          msg.From,
				HasAttachment: msg.HasAttachments,
			}
			if !filters.Match(f, in) {
				continue
			}
			if executeActions(c, body.Folder, msg.UID, f.Actions) {
				applied++
				break
			}
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{"applied": applied})
}

// executeActions carries out a filter's action list against a single message.
// Returns true if any action was applied.
func executeActions(c *imap.Client, folder string, uid uint32, actions []filters.Action) bool {
	for _, a := range actions {
		switch a.Type {
		case "move":
			if a.Value != "" && a.Value != folder {
				c.MoveMessage(folder, uid, a.Value) //nolint:errcheck
				return true
			}
		case "mark_read":
			c.MarkRead(folder, uid, true) //nolint:errcheck
		case "mark_flagged":
			c.MarkFlagged(folder, uid, true) //nolint:errcheck
		case "delete":
			c.DeleteMessage(folder, uid) //nolint:errcheck
			return true
		case "stop":
			return true
		}
	}
	return false
}

// syncFiltersToSieve is a fire-and-forget helper that pushes the current filter
// (and vacation) rules to ManageSieve if it is configured.
func (h *handler) syncFiltersToSieve(username, imapHost, password string) {
	if h.config.SieveHost == "" {
		return
	}
	active, _ := h.rebuildAndUploadSieve(username, imapHost, password)
	activeStr := "false"
	if active {
		activeStr = "true"
	}
	_ = h.settings.Set(username, imapHost, map[string]string{
		"vacation_sieve_active": activeStr,
	})
}
