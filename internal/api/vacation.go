package api

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"github.com/jsuto/letrvu/internal/filters"
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
		sieveActive, sieveErrMsg = h.rebuildAndUploadSieve(sess.Username, sess.IMAPHost, sess.Password)
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

// rebuildAndUploadSieve reads the current vacation and filters settings from DB,
// rebuilds the combined Sieve script, and uploads it to ManageSieve.
// Returns (active, errMsg). active is true when the script was successfully uploaded.
func (h *handler) rebuildAndUploadSieve(username, imapHost, password string) (active bool, errMsg string) {
	c, err := sieve.Connect(h.config.SieveHost, username, password)
	if err != nil {
		log.Printf("sieve: connect failed user=%s host=%s: %v", username, h.config.SieveHost, err)
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

	// Read existing active script content.
	existing := ""
	if activeScript != "" {
		existing, err = c.GetScript(activeScript)
		if err != nil {
			log.Printf("sieve: getscript failed user=%s script=%s: %v", username, activeScript, err)
			return false, "failed to read existing script: " + err.Error()
		}
	}

	// Load vacation settings from DB.
	vals, _ := h.settings.Get(username, imapHost)
	vacEnabled := vals["vacation_enabled"] == "true"
	vacSubject := vals["vacation_subject"]
	vacBody := vals["vacation_body"]
	vacStart := vals["vacation_start"]
	vacEnd := vals["vacation_end"]

	// Load filters from DB.
	var filterList []filters.Filter
	if h.filters != nil {
		filterList, _ = h.filters.List(username, imapHost)
	}

	// Strip both managed blocks so we start clean, preserving user rules.
	base := sieve.RemoveVacation(sieve.RemoveFilters(existing))

	// Re-inject filters block (before vacation so filters run first).
	filtersBlock := filters.BuildSieveBlock(filterList)
	filtersExts := filters.RequiredExtensions(filterList)
	base = sieve.InjectFilters(base, filtersBlock, filtersExts)

	// Re-inject vacation block.
	if vacEnabled {
		base = sieve.InjectVacation(base, vacSubject, vacBody, vacStart, vacEnd, []string{username})
	}

	// If nothing is managed, nothing to upload. Remove our blocks from the
	// active script only if they were there before.
	if !vacEnabled && filtersBlock == "" {
		if existing == base {
			// Nothing changed — no-op.
			return false, ""
		}
		// Cleaned — upload the stripped script.
	}

	target := activeScript
	if target == "" {
		target = "letrvu"
	}

	if err := c.PutScript(target, base); err != nil {
		log.Printf("sieve: putscript failed user=%s: %v", username, err)
		return false, "failed to upload script: " + err.Error()
	}
	if activeScript == "" {
		if err := c.SetActive(target); err != nil {
			log.Printf("sieve: setactive failed user=%s: %v", username, err)
			return false, "failed to activate script: " + err.Error()
		}
	}
	log.Printf("sieve: script rebuilt user=%s script=%s host=%s vacation=%v filters=%d",
		username, target, h.config.SieveHost, vacEnabled, len(filterList))
	return true, ""
}
