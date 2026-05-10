package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/jsuto/letrvu/internal/contacts"
)

func (h *handler) listContacts(w http.ResponseWriter, r *http.Request) {
	sess := h.sessionFrom(r)
	list, err := h.contacts.List(sess.Username, sess.IMAPHost)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResp(err.Error()))
		return
	}
	if list == nil {
		list = []contacts.Contact{}
	}
	writeJSON(w, http.StatusOK, list)
}

func (h *handler) getContact(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResp("invalid id"))
		return
	}
	sess := h.sessionFrom(r)
	c, err := h.contacts.Get(id, sess.Username, sess.IMAPHost)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResp(err.Error()))
		return
	}
	if c == nil {
		writeJSON(w, http.StatusNotFound, errorResp("not found"))
		return
	}
	writeJSON(w, http.StatusOK, c)
}

func (h *handler) createContact(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name   string                  `json:"name"`
		Notes  string                  `json:"notes"`
		Emails []contacts.ContactEmail `json:"emails"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResp("invalid request body"))
		return
	}
	sess := h.sessionFrom(r)
	c, err := h.contacts.Create(sess.Username, sess.IMAPHost, body.Name, body.Notes, body.Emails)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResp(err.Error()))
		return
	}
	writeJSON(w, http.StatusCreated, c)
}

func (h *handler) updateContact(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResp("invalid id"))
		return
	}
	var body struct {
		Name   string                  `json:"name"`
		Notes  string                  `json:"notes"`
		Emails []contacts.ContactEmail `json:"emails"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResp("invalid request body"))
		return
	}
	sess := h.sessionFrom(r)
	c, err := h.contacts.Update(id, sess.Username, sess.IMAPHost, body.Name, body.Notes, body.Emails)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResp(err.Error()))
		return
	}
	if c == nil {
		writeJSON(w, http.StatusNotFound, errorResp("not found"))
		return
	}
	writeJSON(w, http.StatusOK, c)
}

func (h *handler) deleteContact(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResp("invalid id"))
		return
	}
	sess := h.sessionFrom(r)
	if err := h.contacts.Delete(id, sess.Username, sess.IMAPHost); err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResp(err.Error()))
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *handler) autocompleteContacts(w http.ResponseWriter, r *http.Request) {
	prefix := strings.TrimSpace(r.URL.Query().Get("q"))
	if prefix == "" {
		writeJSON(w, http.StatusOK, []contacts.AutocompleteResult{})
		return
	}
	sess := h.sessionFrom(r)
	results, err := h.contacts.Autocomplete(sess.Username, sess.IMAPHost, prefix)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResp(err.Error()))
		return
	}
	if results == nil {
		results = []contacts.AutocompleteResult{}
	}
	writeJSON(w, http.StatusOK, results)
}

func (h *handler) exportContacts(w http.ResponseWriter, r *http.Request) {
	sess := h.sessionFrom(r)
	list, err := h.contacts.List(sess.Username, sess.IMAPHost)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResp(err.Error()))
		return
	}
	vcard := contacts.FormatVCards(list)
	w.Header().Set("Content-Type", "text/vcard; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="contacts.vcf"`)
	w.Write([]byte(vcard)) //nolint:errcheck
}

func (h *handler) importContacts(w http.ResponseWriter, r *http.Request) {
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

	parsed, err := contacts.ParseVCards(file)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResp(err.Error()))
		return
	}

	sess := h.sessionFrom(r)
	var created int
	for _, c := range parsed {
		if _, err := h.contacts.Create(sess.Username, sess.IMAPHost, c.Name, c.Notes, c.Emails); err == nil {
			created++
		}
	}
	writeJSON(w, http.StatusOK, map[string]int{"imported": created})
}

func (h *handler) saveContactFromMessage(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name  string `json:"name"`
		Email string `json:"email"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResp("invalid request body"))
		return
	}
	if body.Email == "" {
		writeJSON(w, http.StatusBadRequest, errorResp("email required"))
		return
	}
	sess := h.sessionFrom(r)
	c, err := h.contacts.SaveFromMessage(sess.Username, sess.IMAPHost, body.Name, body.Email)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResp(err.Error()))
		return
	}
	writeJSON(w, http.StatusOK, c)
}
