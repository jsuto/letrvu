package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/jsuto/letrvu/internal/contacts"
)

func (h *handler) listGroups(w http.ResponseWriter, r *http.Request) {
	sess := h.sessionFrom(r)
	groups, err := h.contacts.ListGroups(sess.Username, sess.IMAPHost)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResp(err.Error()))
		return
	}
	if groups == nil {
		groups = []contacts.ContactGroup{}
	}
	writeJSON(w, http.StatusOK, groups)
}

func (h *handler) createGroup(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Name == "" {
		writeJSON(w, http.StatusBadRequest, errorResp("name required"))
		return
	}
	sess := h.sessionFrom(r)
	g, err := h.contacts.CreateGroup(sess.Username, sess.IMAPHost, body.Name)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResp(err.Error()))
		return
	}
	writeJSON(w, http.StatusCreated, g)
}

func (h *handler) updateGroup(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResp("invalid id"))
		return
	}
	var body struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Name == "" {
		writeJSON(w, http.StatusBadRequest, errorResp("name required"))
		return
	}
	sess := h.sessionFrom(r)
	g, err := h.contacts.UpdateGroup(id, sess.Username, sess.IMAPHost, body.Name)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResp(err.Error()))
		return
	}
	if g == nil {
		writeJSON(w, http.StatusNotFound, errorResp("not found"))
		return
	}
	writeJSON(w, http.StatusOK, g)
}

func (h *handler) deleteGroup(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResp("invalid id"))
		return
	}
	sess := h.sessionFrom(r)
	if err := h.contacts.DeleteGroup(id, sess.Username, sess.IMAPHost); err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResp(err.Error()))
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *handler) addGroupMember(w http.ResponseWriter, r *http.Request) {
	groupID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResp("invalid group id"))
		return
	}
	var body struct {
		ContactID int64 `json:"contact_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.ContactID == 0 {
		writeJSON(w, http.StatusBadRequest, errorResp("contact_id required"))
		return
	}
	sess := h.sessionFrom(r)
	if err := h.contacts.AddGroupMember(groupID, body.ContactID, sess.Username, sess.IMAPHost); err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResp(err.Error()))
		return
	}
	g, err := h.contacts.GetGroup(groupID, sess.Username, sess.IMAPHost)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResp(err.Error()))
		return
	}
	writeJSON(w, http.StatusOK, g)
}

func (h *handler) removeGroupMember(w http.ResponseWriter, r *http.Request) {
	groupID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResp("invalid group id"))
		return
	}
	contactID, err := strconv.ParseInt(r.PathValue("contact_id"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResp("invalid contact id"))
		return
	}
	sess := h.sessionFrom(r)
	if err := h.contacts.RemoveGroupMember(groupID, contactID, sess.Username, sess.IMAPHost); err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResp(err.Error()))
		return
	}
	g, err := h.contacts.GetGroup(groupID, sess.Username, sess.IMAPHost)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResp(err.Error()))
		return
	}
	writeJSON(w, http.StatusOK, g)
}
