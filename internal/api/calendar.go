package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/jsuto/letrvu/internal/calendar"
)

func (h *handler) listCalendarEvents(w http.ResponseWriter, r *http.Request) {
	sess := h.sessionFrom(r)

	from, to := monthRange(r)

	events, err := h.calendar.List(sess.Username, sess.IMAPHost, from, to)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResp(err.Error()))
		return
	}
	writeJSON(w, http.StatusOK, events)
}

func (h *handler) createCalendarEvent(w http.ResponseWriter, r *http.Request) {
	var ev calendar.Event
	if err := json.NewDecoder(r.Body).Decode(&ev); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResp("invalid request body"))
		return
	}
	sess := h.sessionFrom(r)
	created, err := h.calendar.Create(sess.Username, sess.IMAPHost, ev)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResp(err.Error()))
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

func (h *handler) getCalendarEvent(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResp("invalid id"))
		return
	}
	sess := h.sessionFrom(r)
	ev, err := h.calendar.Get(id, sess.Username, sess.IMAPHost)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResp(err.Error()))
		return
	}
	if ev == nil {
		writeJSON(w, http.StatusNotFound, errorResp("not found"))
		return
	}
	writeJSON(w, http.StatusOK, ev)
}

func (h *handler) updateCalendarEvent(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResp("invalid id"))
		return
	}
	var ev calendar.Event
	if err := json.NewDecoder(r.Body).Decode(&ev); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResp("invalid request body"))
		return
	}
	sess := h.sessionFrom(r)
	updated, err := h.calendar.Update(id, sess.Username, sess.IMAPHost, ev)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResp(err.Error()))
		return
	}
	if updated == nil {
		writeJSON(w, http.StatusNotFound, errorResp("not found"))
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

func (h *handler) deleteCalendarEvent(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResp("invalid id"))
		return
	}
	sess := h.sessionFrom(r)
	if err := h.calendar.Delete(id, sess.Username, sess.IMAPHost); err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResp(err.Error()))
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *handler) exportCalendar(w http.ResponseWriter, r *http.Request) {
	sess := h.sessionFrom(r)
	// Export all events (wide range).
	from := time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2100, 1, 1, 0, 0, 0, 0, time.UTC)
	events, err := h.calendar.List(sess.Username, sess.IMAPHost, from, to)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResp(err.Error()))
		return
	}
	ics := calendar.FormatICal(events)
	w.Header().Set("Content-Type", "text/calendar; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="calendar.ics"`)
	w.Write([]byte(ics)) //nolint:errcheck
}

func (h *handler) importCalendar(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(4 << 20); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResp("invalid multipart form"))
		return
	}
	file, _, err := r.FormFile("file")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResp("missing file field"))
		return
	}
	defer file.Close()

	src, err := calendar.ReadAll(file)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResp("could not read file"))
		return
	}
	events, err := calendar.ParseICal(src)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResp(err.Error()))
		return
	}

	sess := h.sessionFrom(r)
	var imported int
	for _, ev := range events {
		if _, err := h.calendar.Create(sess.Username, sess.IMAPHost, ev); err == nil {
			imported++
		}
	}
	writeJSON(w, http.StatusOK, map[string]int{"imported": imported})
}

// importCalendarFromInvite accepts a raw iCal string (from an email invite)
// and adds the first event found to the calendar.
func (h *handler) importCalendarFromInvite(w http.ResponseWriter, r *http.Request) {
	var body struct {
		ICal string `json:"ical"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResp("invalid request body"))
		return
	}
	events, err := calendar.ParseICal(body.ICal)
	if err != nil || len(events) == 0 {
		writeJSON(w, http.StatusBadRequest, errorResp("no valid event in invite"))
		return
	}
	sess := h.sessionFrom(r)
	created, err := h.calendar.Create(sess.Username, sess.IMAPHost, events[0])
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResp(err.Error()))
		return
	}
	writeJSON(w, http.StatusOK, created)
}

// monthRange parses ?from= and ?to= query params (RFC3339) and falls back to
// the current calendar month if absent.
func monthRange(r *http.Request) (time.Time, time.Time) {
	parse := func(key string) (time.Time, bool) {
		s := r.URL.Query().Get(key)
		if s == "" {
			return time.Time{}, false
		}
		t, err := time.Parse(time.RFC3339, s)
		return t, err == nil
	}
	from, okFrom := parse("from")
	to, okTo := parse("to")
	if okFrom && okTo {
		return from, to
	}
	now := time.Now()
	from = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	to = from.AddDate(0, 1, 0)
	return from, to
}
