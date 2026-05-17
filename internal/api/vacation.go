package api

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"github.com/jsuto/letrvu/internal/sieve"
)

type vacationResp struct {
	Enabled         bool   `json:"enabled"`
	Subject         string `json:"subject"`
	Body            string `json:"body"`
	Start           string `json:"start"`
	End             string `json:"end"`
	SieveConfigured bool   `json:"sieve_configured"`
	SieveActive     bool   `json:"sieve_active"`
	SieveError      string `json:"sieve_error,omitempty"`
}

func (h *handler) getVacation(w http.ResponseWriter, r *http.Request) {
	sess := h.sessionFrom(r)
	vals, err := h.settings.Get(sess.Username, sess.IMAPHost)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResp(err.Error()))
		return
	}
	writeJSON(w, http.StatusOK, vacationResp{
		Enabled:         vals["vacation_enabled"] == "true",
		Subject:         vals["vacation_subject"],
		Body:            vals["vacation_body"],
		Start:           vals["vacation_start"],
		End:             vals["vacation_end"],
		SieveConfigured: h.config.SieveHost != "",
		SieveActive:     vals["vacation_sieve_active"] == "true",
	})
}

func (h *handler) setVacation(w http.ResponseWriter, r *http.Request) {
	sess := h.sessionFrom(r)

	var body struct {
		Enabled bool   `json:"enabled"`
		Subject string `json:"subject"`
		Body    string `json:"body"`
		Start   string `json:"start"`
		End     string `json:"end"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResp("invalid request body"))
		return
	}
	if body.Enabled && strings.TrimSpace(body.Subject) == "" {
		writeJSON(w, http.StatusBadRequest, errorResp("subject is required"))
		return
	}
	if body.Enabled && strings.TrimSpace(body.Body) == "" {
		writeJSON(w, http.StatusBadRequest, errorResp("message body is required"))
		return
	}

	// Always persist locally first, regardless of whether Sieve succeeds.
	enabledStr := "false"
	if body.Enabled {
		enabledStr = "true"
	}
	if err := h.settings.Set(sess.Username, sess.IMAPHost, map[string]string{
		"vacation_enabled": enabledStr,
		"vacation_subject": body.Subject,
		"vacation_body":    body.Body,
		"vacation_start":   body.Start,
		"vacation_end":     body.End,
	}); err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResp("failed to save vacation settings"))
		return
	}

	// Attempt ManageSieve upload only when SIEVE_HOST is configured.
	sieveConfigured := h.config.SieveHost != ""
	var sieveActive bool
	var sieveErrMsg string
	if sieveConfigured {
		sieveActive, sieveErrMsg = uploadVacationSieve(
			h.config.SieveHost, sess.Username, sess.Password,
			body.Enabled, body.Subject, body.Body, body.Start, body.End,
		)
	}

	sieveActiveStr := "false"
	if sieveActive {
		sieveActiveStr = "true"
	}
	_ = h.settings.Set(sess.Username, sess.IMAPHost, map[string]string{
		"vacation_sieve_active": sieveActiveStr,
	})

	log.Printf("audit: vacation_set user=%s ip=%s enabled=%v sieve_configured=%v sieve_active=%v",
		sess.Username, h.clientIP(r), body.Enabled, sieveConfigured, sieveActive)

	writeJSON(w, http.StatusOK, vacationResp{
		Enabled:         body.Enabled,
		Subject:         body.Subject,
		Body:            body.Body,
		Start:           body.Start,
		End:             body.End,
		SieveConfigured: sieveConfigured,
		SieveActive:     sieveActive,
		SieveError:      sieveErrMsg,
	})
}

// uploadVacationSieve connects to ManageSieve and uploads or deactivates the
// vacation script. Returns (true, "") on success, (false, reason) on failure.
func uploadVacationSieve(sieveHost, username, password string, enabled bool, subject, body, start, end string) (active bool, errMsg string) {
	const scriptName = "letrvu-vacation"

	c, err := sieve.Connect(sieveHost, username, password)
	if err != nil {
		log.Printf("sieve: connect failed user=%s host=%s: %v", username, sieveHost, err)
		return false, "ManageSieve not available: " + err.Error()
	}
	defer c.Close()

	if enabled {
		script := sieve.BuildVacationScript(subject, body, start, end, []string{username})
		if err := c.PutScript(scriptName, script); err != nil {
			log.Printf("sieve: putscript failed user=%s: %v", username, err)
			return false, "failed to upload script: " + err.Error()
		}
		if err := c.SetActive(scriptName); err != nil {
			log.Printf("sieve: setactive failed user=%s: %v", username, err)
			return false, "failed to activate script: " + err.Error()
		}
		log.Printf("sieve: vacation activated user=%s host=%s", username, sieveHost)
		return true, ""
	}

	// Disable: deactivate all scripts without deleting (preserves config for re-enable).
	if err := c.SetActive(""); err != nil {
		log.Printf("sieve: deactivate failed user=%s: %v", username, err)
	} else {
		log.Printf("sieve: vacation deactivated user=%s host=%s", username, sieveHost)
	}
	return false, ""
}
