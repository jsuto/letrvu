package sieve

import (
	"fmt"
	"strings"
)

// BuildVacationScript generates a Sieve script (RFC 5230) that sends a
// vacation autoresponse. start and end are optional "YYYY-MM-DD" strings;
// when both are non-empty the date extension (RFC 5260) bounds the window.
// addresses is the list of addresses the user receives mail at.
func BuildVacationScript(subject, body, start, end string, addresses []string) string {
	var sb strings.Builder

	useDates := start != "" && end != ""

	if useDates {
		sb.WriteString(`require ["vacation", "date", "relational"];` + "\n")
	} else {
		sb.WriteString(`require ["vacation"];` + "\n")
	}

	if useDates {
		fmt.Fprintf(&sb,
			"if allof(\n  currentdate :value \"ge\" \"date\" %s,\n  currentdate :value \"le\" \"date\" %s\n) {\n",
			sieveQuote(start), sieveQuote(end))
	}

	indent := ""
	if useDates {
		indent = "  "
	}

	sb.WriteString(indent + "vacation :days 1")
	if len(addresses) > 0 {
		quoted := make([]string, len(addresses))
		for i, a := range addresses {
			quoted[i] = sieveQuote(a)
		}
		fmt.Fprintf(&sb, " :addresses [%s]", strings.Join(quoted, ", "))
	}
	fmt.Fprintf(&sb, " :subject %s\n", sieveQuote(subject))

	// Use Sieve text: block for the body so newlines are preserved as-is.
	// Lines starting with '.' must be dot-stuffed.
	sb.WriteString(indent + "text:\n")
	lines := strings.Split(strings.ReplaceAll(body, "\r\n", "\n"), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, ".") {
			sb.WriteString(".")
		}
		sb.WriteString(line + "\n")
	}
	sb.WriteString(".\n")
	sb.WriteString(indent + ";\n")

	if useDates {
		sb.WriteString("}\n")
	}

	return sb.String()
}

// sieveQuote returns a Sieve quoted string with proper escaping.
func sieveQuote(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	return `"` + s + `"`
}
