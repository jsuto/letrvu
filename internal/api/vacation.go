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

// uploadVacationSieve connects to ManageSieve and injects or removes the
// vacation rule from the user's active script. The existing script content
// (fileinto rules, etc.) is preserved. Returns (true, "") on success.
func uploadVacationSieve(sieveHost, username, password string, enabled bool, subject, body, start, end string) (active bool, errMsg string) {
	c, err := sieve.Connect(sieveHost, username, password)
	if err != nil {
		log.Printf("sieve: connect failed user=%s host=%s: %v", username, sieveHost, err)
		return false, "ManageSieve not available: " + err.Error()
	}
	defer c.Close()

	scripts, err := c.ListScripts()
	if err != nil {
		log.Printf("sieve: listscripts failed user=%s: %v", username, err)
		return false, "failed to list scripts: " + err.Error()
	}

	// Find the currently active script.
	activeScript := ""
	for _, s := range scripts {
		if s.Active {
			activeScript = s.Name
			break
		}
	}

	if enabled {
		// Read existing active script content (empty string if none).
		existing := ""
		if activeScript != "" {
			existing, err = c.GetScript(activeScript)
			if err != nil {
				log.Printf("sieve: getscript failed user=%s script=%s: %v", username, activeScript, err)
				return false, "failed to read existing script: " + err.Error()
			}
		}

		// Write to the active script so fileinto rules stay in place.
		// Fall back to a new "letrvu-vacation" script if nothing is active.
		target := activeScript
		if target == "" {
			target = "letrvu-vacation"
		}

		merged := sieve.InjectVacation(existing, subject, body, start, end, []string{username})
		if err := c.PutScript(target, merged); err != nil {
			log.Printf("sieve: putscript failed user=%s: %v", username, err)
			return false, "failed to upload script: " + err.Error()
		}
		// Activate only if nothing was active before (i.e. we created a new script).
		if activeScript == "" {
			if err := c.SetActive(target); err != nil {
				log.Printf("sieve: setactive failed user=%s: %v", username, err)
				return false, "failed to activate script: " + err.Error()
			}
		}
		log.Printf("sieve: vacation injected user=%s script=%s host=%s", username, target, sieveHost)
		return true, ""
	}

	// Disabling: remove the vacation markers from the active script.
	if activeScript == "" {
		log.Printf("sieve: no active script found, nothing to remove user=%s", username)
		return false, ""
	}

	existing, err := c.GetScript(activeScript)
	if err != nil {
		log.Printf("sieve: getscript failed user=%s: %v", username, err)
		return false, "failed to read script: " + err.Error()
	}

	cleaned := sieve.RemoveVacation(existing)
	if cleaned == existing {
		// No letrvu markers found. This is the old "letrvu-vacation" standalone
		// script from before the injection approach. Restore the previous script
		// by finding any other non-vacation script and activating it.
		if activeScript == "letrvu-vacation" {
			for _, s := range scripts {
				if s.Name != "letrvu-vacation" {
					if err := c.SetActive(s.Name); err != nil {
						log.Printf("sieve: setactive fallback failed user=%s script=%s: %v", username, s.Name, err)
					} else {
						log.Printf("sieve: restored previous script %s user=%s host=%s", s.Name, username, sieveHost)
					}
					break
				}
			}
		}
		return false, ""
	}

	if err := c.PutScript(activeScript, cleaned); err != nil {
		log.Printf("sieve: putscript failed user=%s: %v", username, err)
		return false, "failed to update script: " + err.Error()
	}
	log.Printf("sieve: vacation removed user=%s script=%s host=%s", username, activeScript, sieveHost)
	return false, ""
}
