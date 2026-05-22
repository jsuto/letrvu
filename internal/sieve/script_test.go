package sieve

import (
	"strings"
	"testing"
)

func TestBuildVacationScript_NoDateRange(t *testing.T) {
	s := BuildVacationScript("Out of office", "I am away.", "", "", []string{"user@example.com"})

	if !strings.HasPrefix(s, `require ["vacation"];`) {
		t.Errorf("expected simple require, got: %q", s[:min(len(s), 40)])
	}
	if strings.Contains(s, "date") || strings.Contains(s, "relational") {
		t.Error("should not include date/relational extensions when no date range given")
	}
	if strings.Contains(s, "if allof") {
		t.Error("should not include date guard when no date range given")
	}
	if !strings.Contains(s, `:subject "Out of office"`) {
		t.Error("missing :subject")
	}
	if !strings.Contains(s, `:addresses ["user@example.com"]`) {
		t.Error("missing :addresses")
	}
	if !strings.Contains(s, "I am away.") {
		t.Error("missing body text")
	}
}

func TestBuildVacationScript_WithDateRange(t *testing.T) {
	s := BuildVacationScript("On leave", "Back soon.", "2026-06-01", "2026-06-15", []string{"user@example.com"})

	if !strings.Contains(s, `require ["vacation", "date", "relational"];`) {
		t.Error("missing date/relational require")
	}
	if !strings.Contains(s, `currentdate :value "ge" "date" "2026-06-01"`) {
		t.Error("missing ge date condition")
	}
	if !strings.Contains(s, `currentdate :value "le" "date" "2026-06-15"`) {
		t.Error("missing le date condition")
	}
	if !strings.Contains(s, "if allof(") {
		t.Error("missing if allof guard")
	}
	// Script must close the if block.
	if !strings.Contains(s, "}\n") {
		t.Error("missing closing brace for if block")
	}
}

func TestBuildVacationScript_OnlyStartDate(t *testing.T) {
	// A date range requires both start AND end; with only one, no guard is emitted.
	s := BuildVacationScript("Away", "Back soon.", "2026-06-01", "", []string{"user@example.com"})

	if strings.Contains(s, "if allof") {
		t.Error("should not emit date guard when only start date is set")
	}
}

func TestBuildVacationScript_MultipleAddresses(t *testing.T) {
	s := BuildVacationScript("OOO", "Away.", "", "", []string{"a@example.com", "b@example.com"})

	if !strings.Contains(s, `"a@example.com", "b@example.com"`) {
		t.Errorf("expected both addresses quoted, got:\n%s", s)
	}
}

func TestBuildVacationScript_NoAddresses(t *testing.T) {
	s := BuildVacationScript("OOO", "Away.", "", "", nil)

	if strings.Contains(s, ":addresses") {
		t.Error("should not emit :addresses when list is empty")
	}
}

func TestBuildVacationScript_DotStuffing(t *testing.T) {
	// A line starting with '.' in the body must be dot-stuffed so it doesn't
	// prematurely terminate the Sieve text: block.
	body := "Hello.\n. this line starts with a dot\nGoodbye."
	s := BuildVacationScript("OOO", body, "", "", nil)

	if !strings.Contains(s, ".. this line starts with a dot") {
		t.Errorf("line starting with '.' was not dot-stuffed:\n%s", s)
	}
}

func TestBuildVacationScript_SpecialCharsInSubject(t *testing.T) {
	s := BuildVacationScript(`Out of "office"`, "Away.", "", "", nil)

	if !strings.Contains(s, `"Out of \"office\""`) {
		t.Errorf("double quotes in subject not escaped:\n%s", s)
	}
}

func TestBuildVacationScript_BackslashInSubject(t *testing.T) {
	s := BuildVacationScript(`Away\gone`, "Away.", "", "", nil)

	if !strings.Contains(s, `"Away\\gone"`) {
		t.Errorf("backslash in subject not escaped:\n%s", s)
	}
}

func TestBuildVacationScript_TextBlockTerminator(t *testing.T) {
	s := BuildVacationScript("OOO", "Body text.", "", "", nil)

	// The text: block must be terminated by a lone '.' on its own line.
	if !strings.Contains(s, "\n.\n") {
		t.Errorf("text block terminator '.' not found:\n%s", s)
	}
	// The command must end with ';'
	if !strings.Contains(s, ";\n") {
		t.Errorf("missing command terminator ';':\n%s", s)
	}
}
