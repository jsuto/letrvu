package sieve

import (
	"strings"
	"testing"
)

func TestInjectFilters_EmptyScript(t *testing.T) {
	block := "if header :contains \"Subject\" \"test\" {\n  fileinto \"Test\";\n}\n"
	got := InjectFilters("", block, []string{"fileinto"})
	if !strings.Contains(got, filtersMarkerStart) {
		t.Error("expected filters marker start")
	}
	if !strings.Contains(got, "fileinto") {
		t.Error("expected fileinto in output")
	}
	if !strings.Contains(got, `require ["fileinto"]`) {
		t.Error("expected require statement")
	}
}

func TestInjectFilters_PreservesExistingRules(t *testing.T) {
	existing := `require ["fileinto"];
# my custom rule
if header :contains "Subject" "newsletter" {
  fileinto "Newsletters";
}
`
	block := "if header :contains \"From\" \"boss@\" {\n  addflag \"\\\\Flagged\";\n}\n"
	got := InjectFilters(existing, block, []string{"imap4flags"})
	if !strings.Contains(got, "newsletter") {
		t.Error("existing rule should be preserved")
	}
	if !strings.Contains(got, "boss@") {
		t.Error("new block should be present")
	}
}

func TestRemoveFilters_Clean(t *testing.T) {
	existing := `require ["fileinto"];
# custom rule
if true { fileinto "X"; }

` + filtersMarkerStart + `
if header :contains "Subject" "x" { discard; }
` + filtersMarkerEnd + `
`
	got := RemoveFilters(existing)
	if strings.Contains(got, filtersMarkerStart) {
		t.Error("marker should be removed")
	}
	if strings.Contains(got, "discard") {
		t.Error("injected block should be removed")
	}
	if !strings.Contains(got, "custom rule") {
		t.Error("existing rules should be preserved")
	}
}

func TestInjectFilters_EmptyBlockRemoves(t *testing.T) {
	existing := `require ["fileinto"];

` + filtersMarkerStart + `
if header :contains "Subject" "x" { discard; }
` + filtersMarkerEnd + `
`
	got := InjectFilters(existing, "", nil)
	if strings.Contains(got, filtersMarkerStart) {
		t.Error("empty block should remove markers")
	}
}

func TestInjectFilters_Replace(t *testing.T) {
	existing := `require ["fileinto"];

` + filtersMarkerStart + `
if header :contains "Subject" "old" { fileinto "Old"; }
` + filtersMarkerEnd + `
`
	newBlock := "if header :contains \"Subject\" \"new\" { fileinto \"New\"; }\n"
	got := InjectFilters(existing, newBlock, []string{"fileinto"})
	if strings.Contains(got, "Old") {
		t.Error("old block should be replaced")
	}
	if !strings.Contains(got, "New") {
		t.Error("new block should be present")
	}
}
