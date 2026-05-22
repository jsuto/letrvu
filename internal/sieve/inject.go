package sieve

import (
	"regexp"
	"strings"
)

const (
	markerStart = "# === letrvu-vacation-start ==="
	markerEnd   = "# === letrvu-vacation-end ==="
)

// InjectVacation adds the vacation rule into an existing Sieve script.
// It merges require statements and appends the vacation block between markers.
// If the script already contains a letrvu vacation block it is replaced,
// so calling this again on an already-injected script is safe.
func InjectVacation(existing, subject, body, start, end string, addresses []string) string {
	// Strip any previous injection so we start clean.
	cleaned := RemoveVacation(existing)

	useDates := start != "" && end != ""
	newReqs := vacationRequires(useDates)

	withReqs := mergeRequiresInScript(cleaned, newReqs)

	block := vacationActionBlock(subject, body, start, end, addresses)
	return strings.TrimRight(withReqs, "\n") + "\n\n" +
		markerStart + "\n" + block + markerEnd + "\n"
}

// RemoveVacation strips the letrvu-injected vacation block from a script,
// leaving all other rules intact. Returns the script unchanged if no markers
// are found.
func RemoveVacation(script string) string {
	si := strings.Index(script, markerStart)
	ei := strings.Index(script, markerEnd)
	if si < 0 || ei < 0 || ei <= si {
		return script
	}
	before := strings.TrimRight(script[:si], "\n")
	after := strings.TrimLeft(script[ei+len(markerEnd):], "\n")
	if after == "" {
		return before + "\n"
	}
	return before + "\n" + after
}

// mergeRequiresInScript adds exts to the script's require statement,
// creating one at the top if none exists.
func mergeRequiresInScript(script string, exts []string) string {
	if len(exts) == 0 {
		return script
	}
	re := regexp.MustCompile(`(?m)^require\s+(?:"[^"]*"|\[[^\]]*\])\s*;`)
	loc := re.FindStringIndex(script)
	if loc == nil {
		return buildRequireStatement(exts) + "\n" + script
	}

	existing := script[loc[0]:loc[1]]
	merged := parseRequireExts(existing)
	for _, ext := range exts {
		found := false
		for _, e := range merged {
			if e == ext {
				found = true
				break
			}
		}
		if !found {
			merged = append(merged, ext)
		}
	}
	return script[:loc[0]] + buildRequireStatement(merged) + script[loc[1]:]
}

// parseRequireExts extracts extension names from a Sieve require statement.
func parseRequireExts(req string) []string {
	re := regexp.MustCompile(`"([^"]+)"`)
	matches := re.FindAllStringSubmatch(req, -1)
	var exts []string
	for _, m := range matches {
		exts = append(exts, m[1])
	}
	return exts
}

// buildRequireStatement builds a Sieve require statement from a list of extensions.
func buildRequireStatement(exts []string) string {
	if len(exts) == 0 {
		return ""
	}
	quoted := make([]string, len(exts))
	for i, e := range exts {
		quoted[i] = `"` + e + `"`
	}
	return "require [" + strings.Join(quoted, ", ") + "];"
}
