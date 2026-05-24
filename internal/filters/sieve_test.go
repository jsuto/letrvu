package filters

import (
	"strings"
	"testing"
)

func TestBuildSieveBlock_Empty(t *testing.T) {
	got := BuildSieveBlock(nil)
	if got != "" {
		t.Errorf("expected empty, got %q", got)
	}
}

func TestBuildSieveBlock_MoveRule(t *testing.T) {
	f := Filter{
		Enabled: true,
		Name:    "Invoices",
		MatchAll: true,
		Conditions: []Condition{{Field: "subject", Op: "contains", Value: "invoice"}},
		Actions:    []Action{{Type: "move", Value: "Invoices"}},
	}
	got := BuildSieveBlock([]Filter{f})
	if !strings.Contains(got, `header :contains "Subject" "invoice"`) {
		t.Errorf("missing header test in: %s", got)
	}
	if !strings.Contains(got, `fileinto "Invoices"`) {
		t.Errorf("missing fileinto in: %s", got)
	}
}

func TestBuildSieveBlock_MultiConditionAnd(t *testing.T) {
	f := Filter{
		Enabled:  true,
		Name:     "Multi",
		MatchAll: true,
		Conditions: []Condition{
			{Field: "from", Op: "contains", Value: "boss@"},
			{Field: "subject", Op: "contains", Value: "urgent"},
		},
		Actions: []Action{{Type: "mark_flagged"}},
	}
	got := BuildSieveBlock([]Filter{f})
	if !strings.Contains(got, "allof(") {
		t.Errorf("expected allof, got: %s", got)
	}
}

func TestBuildSieveBlock_MultiConditionOr(t *testing.T) {
	f := Filter{
		Enabled:  true,
		Name:     "Multi",
		MatchAll: false,
		Conditions: []Condition{
			{Field: "from", Op: "contains", Value: "a@"},
			{Field: "from", Op: "contains", Value: "b@"},
		},
		Actions: []Action{{Type: "mark_read"}},
	}
	got := BuildSieveBlock([]Filter{f})
	if !strings.Contains(got, "anyof(") {
		t.Errorf("expected anyof, got: %s", got)
	}
}

func TestRequiredExtensions(t *testing.T) {
	fs := []Filter{
		{Enabled: true, Actions: []Action{{Type: "move", Value: "Folder"}}},
		{Enabled: true, Actions: []Action{{Type: "mark_read"}}},
	}
	exts := RequiredExtensions(fs)
	hasFileinto := false
	hasFlags := false
	for _, e := range exts {
		if e == "fileinto" {
			hasFileinto = true
		}
		if e == "imap4flags" {
			hasFlags = true
		}
	}
	if !hasFileinto {
		t.Error("expected fileinto extension")
	}
	if !hasFlags {
		t.Error("expected imap4flags extension")
	}
}

func TestBuildSieveBlock_DisabledSkipped(t *testing.T) {
	f := Filter{
		Enabled:    false,
		Conditions: []Condition{{Field: "subject", Op: "contains", Value: "x"}},
		Actions:    []Action{{Type: "mark_read"}},
	}
	got := BuildSieveBlock([]Filter{f})
	if got != "" {
		t.Errorf("disabled filter should produce no output, got: %s", got)
	}
}
