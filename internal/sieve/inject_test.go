package sieve

import (
	"strings"
	"testing"
)

const fileintoScript = `require "fileinto";
if header :contains "X-Spam" "Yes" {
  fileinto "Junk";
}
`

func TestInjectVacation_EmptyScript(t *testing.T) {
	s := InjectVacation("", "OOO", "Away.", "", "", []string{"u@example.com"})

	if !strings.Contains(s, `require ["vacation"]`) {
		t.Error("missing vacation require")
	}
	if !strings.Contains(s, markerStart) || !strings.Contains(s, markerEnd) {
		t.Error("missing letrvu markers")
	}
	if !strings.Contains(s, "vacation :days 1") {
		t.Error("missing vacation action")
	}
}

func TestInjectVacation_PreservesExistingRules(t *testing.T) {
	s := InjectVacation(fileintoScript, "OOO", "Away.", "", "", []string{"u@example.com"})

	if !strings.Contains(s, `fileinto "Junk"`) {
		t.Error("existing fileinto rule was lost")
	}
	if !strings.Contains(s, "vacation :days 1") {
		t.Error("missing vacation action")
	}
}

func TestInjectVacation_MergesRequires(t *testing.T) {
	s := InjectVacation(fileintoScript, "OOO", "Away.", "", "", nil)

	// fileinto and vacation must both be present in a single require.
	if !strings.Contains(s, `"fileinto"`) {
		t.Error("fileinto extension dropped from require")
	}
	if !strings.Contains(s, `"vacation"`) {
		t.Error("vacation extension missing from require")
	}
	// Must not have duplicate require statements.
	if strings.Count(s, "require") > 1 {
		t.Errorf("duplicate require statements:\n%s", s)
	}
}

func TestInjectVacation_WithDateRange_MergesRequires(t *testing.T) {
	s := InjectVacation(fileintoScript, "OOO", "Away.", "2026-06-01", "2026-06-15", nil)

	if !strings.Contains(s, `"fileinto"`) {
		t.Error("fileinto dropped")
	}
	if !strings.Contains(s, `"vacation"`) {
		t.Error("vacation missing")
	}
	if !strings.Contains(s, `"date"`) {
		t.Error("date extension missing")
	}
	if !strings.Contains(s, `"relational"`) {
		t.Error("relational extension missing")
	}
	if strings.Count(s, "require") > 1 {
		t.Error("duplicate require statements")
	}
}

func TestInjectVacation_ReplacesExistingInjection(t *testing.T) {
	first := InjectVacation(fileintoScript, "OOO v1", "Away v1.", "", "", nil)
	second := InjectVacation(first, "OOO v2", "Away v2.", "", "", nil)

	if strings.Contains(second, "Away v1") {
		t.Error("old vacation body still present after re-inject")
	}
	if !strings.Contains(second, "Away v2") {
		t.Error("new vacation body not found")
	}
	if strings.Count(second, markerStart) != 1 {
		t.Errorf("expected exactly one marker start, got %d", strings.Count(second, markerStart))
	}
	if !strings.Contains(second, `fileinto "Junk"`) {
		t.Error("fileinto rule lost after re-inject")
	}
}

func TestRemoveVacation_WithMarkers(t *testing.T) {
	injected := InjectVacation(fileintoScript, "OOO", "Away.", "", "", nil)
	restored := RemoveVacation(injected)

	if strings.Contains(restored, "vacation :days") {
		t.Error("vacation action still present after removal")
	}
	if strings.Contains(restored, markerStart) || strings.Contains(restored, markerEnd) {
		t.Error("markers still present after removal")
	}
	if !strings.Contains(restored, `fileinto "Junk"`) {
		t.Error("fileinto rule was removed along with vacation")
	}
}

func TestRemoveVacation_WithoutMarkers(t *testing.T) {
	// Removing from a script that has no markers is a no-op.
	result := RemoveVacation(fileintoScript)
	if result != fileintoScript {
		t.Errorf("script changed even though no markers were present:\ngot: %q", result)
	}
}

func TestMergeRequires_NoExistingRequire(t *testing.T) {
	script := "if true { stop; }\n"
	result := mergeRequiresInScript(script, []string{"vacation"})

	if !strings.HasPrefix(result, `require ["vacation"];`) {
		t.Errorf("require not prepended, got: %q", result[:min(len(result), 50)])
	}
}

func TestMergeRequires_SingleQuotedExisting(t *testing.T) {
	script := `require "fileinto";` + "\nif true { stop; }\n"
	result := mergeRequiresInScript(script, []string{"vacation"})

	if strings.Count(result, "require") != 1 {
		t.Error("expected exactly one require statement")
	}
	if !strings.Contains(result, `"fileinto"`) || !strings.Contains(result, `"vacation"`) {
		t.Errorf("missing extension in merged require: %q", result)
	}
}

func TestMergeRequires_NoDuplicates(t *testing.T) {
	script := `require ["vacation"];` + "\nvacation :days 1 \"OOO\" \"body\";\n"
	result := mergeRequiresInScript(script, []string{"vacation"})

	if strings.Count(result, `"vacation"`) != 1 {
		t.Errorf("vacation extension duplicated: %q", result)
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
