package filters

import (
	"fmt"
	"strings"
)

// RequiredExtensions returns the Sieve extensions needed by the filter list.
func RequiredExtensions(filters []Filter) []string {
	need := map[string]bool{}
	for _, f := range filters {
		if !f.Enabled {
			continue
		}
		for _, a := range f.Actions {
			switch a.Type {
			case "move":
				need["fileinto"] = true
			case "mark_read", "mark_flagged":
				need["imap4flags"] = true
			}
		}
	}
	var out []string
	for k := range need {
		out = append(out, k)
	}
	return out
}

// BuildSieveBlock generates the Sieve script fragment for all enabled filters.
// The returned string is meant to be injected between letrvu-filters markers.
func BuildSieveBlock(filters []Filter) string {
	var sb strings.Builder
	for _, f := range filters {
		if !f.Enabled || len(f.Conditions) == 0 || len(f.Actions) == 0 {
			continue
		}
		writeRule(&sb, f)
	}
	return sb.String()
}

func writeRule(sb *strings.Builder, f Filter) {
	// Build the condition test.
	tests := make([]string, 0, len(f.Conditions))
	for _, c := range f.Conditions {
		t := conditionToSieve(c)
		if t != "" {
			tests = append(tests, t)
		}
	}
	if len(tests) == 0 {
		return
	}

	var ifExpr string
	if len(tests) == 1 {
		ifExpr = tests[0]
	} else if f.MatchAll {
		ifExpr = "allof(\n    " + strings.Join(tests, ",\n    ") + "\n  )"
	} else {
		ifExpr = "anyof(\n    " + strings.Join(tests, ",\n    ") + "\n  )"
	}

	if f.Name != "" {
		fmt.Fprintf(sb, "# filter: %s\n", f.Name)
	}
	fmt.Fprintf(sb, "if %s {\n", ifExpr)
	for _, a := range f.Actions {
		writeAction(sb, a)
	}
	fmt.Fprintf(sb, "}\n")
}

func writeAction(sb *strings.Builder, a Action) {
	switch a.Type {
	case "move":
		fmt.Fprintf(sb, "  fileinto %s;\n", sieveQuote(a.Value))
	case "mark_read":
		fmt.Fprintf(sb, "  addflag \"\\\\Seen\";\n")
	case "mark_flagged":
		fmt.Fprintf(sb, "  addflag \"\\\\Flagged\";\n")
	case "delete":
		fmt.Fprintf(sb, "  discard;\n")
	case "stop":
		fmt.Fprintf(sb, "  stop;\n")
	}
}

func conditionToSieve(c Condition) string {
	switch c.Field {
	case "subject":
		return headerTest("Subject", c.Op, c.Value)
	case "from":
		return headerTest("From", c.Op, c.Value)
	case "to":
		return headerTest("To", c.Op, c.Value)
	case "body":
		return bodyTest(c.Op, c.Value)
	case "has_attachment":
		// Sieve does not have a standard attachment test; omit (handled by Go engine only)
		return ""
	}
	return ""
}

func headerTest(header, op, value string) string {
	switch op {
	case "contains":
		return fmt.Sprintf("header :contains %s %s", sieveQuote(header), sieveQuote(value))
	case "not_contains":
		return fmt.Sprintf("not header :contains %s %s", sieveQuote(header), sieveQuote(value))
	case "equals":
		return fmt.Sprintf("header :is %s %s", sieveQuote(header), sieveQuote(value))
	case "not_equals":
		return fmt.Sprintf("not header :is %s %s", sieveQuote(header), sieveQuote(value))
	case "matches":
		return fmt.Sprintf("header :matches %s %s", sieveQuote(header), sieveQuote(value))
	}
	return ""
}

func bodyTest(op, value string) string {
	switch op {
	case "contains":
		return fmt.Sprintf("body :contains %s", sieveQuote(value))
	case "not_contains":
		return fmt.Sprintf("not body :contains %s", sieveQuote(value))
	case "equals":
		return fmt.Sprintf("body :is %s", sieveQuote(value))
	case "not_equals":
		return fmt.Sprintf("not body :is %s", sieveQuote(value))
	case "matches":
		return fmt.Sprintf("body :matches %s", sieveQuote(value))
	}
	return ""
}

func sieveQuote(s string) string {
	return `"` + strings.ReplaceAll(s, `"`, `\"`) + `"`
}
