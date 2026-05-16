package api

import (
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// getPGPKey returns the user's stored encrypted private key blob, if any.
func (h *handler) getPGPKey(w http.ResponseWriter, r *http.Request) {
	sess := h.sessionFrom(r)
	vals, err := h.settings.Get(sess.Username, sess.IMAPHost)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResp("failed to load key"))
		return
	}
	key := vals["pgp_key_enc"]
	if key == "" {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"key": key})
}

// setPGPKey stores the user's passphrase-protected armored private key.
func (h *handler) setPGPKey(w http.ResponseWriter, r *http.Request) {
	sess := h.sessionFrom(r)
	var body struct {
		Key string `json:"key"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || strings.TrimSpace(body.Key) == "" {
		writeJSON(w, http.StatusBadRequest, errorResp("key is required"))
		return
	}
	if err := h.settings.Set(sess.Username, sess.IMAPHost, map[string]string{"pgp_key_enc": body.Key}); err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResp("failed to save key"))
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// deletePGPKey removes the user's stored private key.
func (h *handler) deletePGPKey(w http.ResponseWriter, r *http.Request) {
	sess := h.sessionFrom(r)
	if err := h.settings.Delete(sess.Username, sess.IMAPHost, "pgp_key_enc"); err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResp("failed to delete key"))
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// getContactPGPKey returns the PGP public key for a contact.
func (h *handler) getContactPGPKey(w http.ResponseWriter, r *http.Request) {
	sess := h.sessionFrom(r)
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResp("invalid id"))
		return
	}
	key, err := h.contacts.GetPGPKey(id, sess.Username, sess.IMAPHost)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResp("db error"))
		return
	}
	if key == "" {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"key": key})
}

// setContactPGPKey stores an armored PGP public key for a contact.
func (h *handler) setContactPGPKey(w http.ResponseWriter, r *http.Request) {
	sess := h.sessionFrom(r)
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResp("invalid id"))
		return
	}
	var body struct {
		Key string `json:"key"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResp("invalid body"))
		return
	}
	if err := h.contacts.SetPGPKey(id, sess.Username, sess.IMAPHost, body.Key); err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResp("failed to save key"))
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// deleteContactPGPKey removes the PGP public key from a contact.
func (h *handler) deleteContactPGPKey(w http.ResponseWriter, r *http.Request) {
	sess := h.sessionFrom(r)
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResp("invalid id"))
		return
	}
	if err := h.contacts.SetPGPKey(id, sess.Username, sess.IMAPHost, ""); err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResp("failed to remove key"))
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// getKeyForEmail returns the stored PGP public key for a given email address.
// Used by ComposeModal to check which recipients have stored keys.
func (h *handler) getKeyForEmail(w http.ResponseWriter, r *http.Request) {
	sess := h.sessionFrom(r)
	email := r.URL.Query().Get("email")
	if email == "" {
		writeJSON(w, http.StatusBadRequest, errorResp("email required"))
		return
	}
	key, err := h.contacts.FindPublicKeyByEmail(sess.Username, sess.IMAPHost, email)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResp("db error"))
		return
	}
	if key == "" {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"key": key})
}

// wkdLookup fetches a PGP public key from WKD for the given email address.
// Tries the "advanced" method (domain/.well-known/openpgpkey) first, then the
// "direct" method (openpgpkey.domain/.well-known/openpgpkey/domain) as fallback.
// Returns base64-encoded binary OpenPGP packets which openpgp.js can read via
// readKey({ binaryKey: ... }).
func (h *handler) wkdLookup(w http.ResponseWriter, r *http.Request) {
	email := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("email")))
	at := strings.LastIndex(email, "@")
	if at < 1 {
		writeJSON(w, http.StatusBadRequest, errorResp("invalid email"))
		return
	}
	localpart := email[:at]
	domain := email[at+1:]

	hash := wkdHash(localpart)
	escaped := url.QueryEscape(localpart)

	client := &http.Client{Timeout: 5 * time.Second}

	// Advanced method
	advURL := fmt.Sprintf("https://%s/.well-known/openpgpkey/hu/%s?l=%s", domain, hash, escaped)
	if data, ok := fetchWKD(client, advURL); ok {
		writeJSON(w, http.StatusOK, map[string]string{"key_b64": base64.StdEncoding.EncodeToString(data)})
		return
	}

	// Direct method
	dirURL := fmt.Sprintf("https://openpgpkey.%s/.well-known/openpgpkey/%s/hu/%s?l=%s", domain, domain, hash, escaped)
	if data, ok := fetchWKD(client, dirURL); ok {
		writeJSON(w, http.StatusOK, map[string]string{"key_b64": base64.StdEncoding.EncodeToString(data)})
		return
	}

	w.WriteHeader(http.StatusNotFound)
}

func fetchWKD(client *http.Client, wkdURL string) ([]byte, bool) {
	resp, err := client.Get(wkdURL)
	if err != nil {
		return nil, false
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, false
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, 128*1024))
	if err != nil || len(data) == 0 {
		return nil, false
	}
	return data, true
}

// wkdHash returns the z-base-32 encoding of the SHA-1 hash of the lowercase
// localpart, as required by the WKD URL scheme (RFC draft-koch-openpgp-webkey-service).
func wkdHash(localpart string) string {
	h := sha1.Sum([]byte(strings.ToLower(localpart)))
	return zbase32Encode(h[:])
}

// zbase32Alphabet is the z-base-32 encoding alphabet.
const zbase32Alphabet = "ybndrfg8ejkmcpqxot1uwisza345h769"

func zbase32Encode(data []byte) string {
	result := make([]byte, 0, (len(data)*8+4)/5)
	buf, bits := 0, 0
	for _, b := range data {
		buf = (buf << 8) | int(b)
		bits += 8
		for bits >= 5 {
			bits -= 5
			result = append(result, zbase32Alphabet[(buf>>bits)&0x1f])
		}
	}
	if bits > 0 {
		result = append(result, zbase32Alphabet[(buf<<(5-bits))&0x1f])
	}
	return string(result)
}
