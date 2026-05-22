package sieve

import (
	"fmt"
	"strings"
)

// BuildVacationScript generates a complete standalone Sieve vacation script.
// Use InjectVacation instead when adding vacation to an existing script.
func BuildVacationScript(subject, body, start, end string, addresses []string) string {
	useDates := start != "" && end != ""
	return buildRequireStatement(vacationRequires(useDates)) + "\n" +
		vacationActionBlock(subject, body, start, end, addresses)
}

// vacationRequires returns the Sieve extensions needed for the vacation rule.
func vacationRequires(useDates bool) []string {
	if useDates {
		return []string{"vacation", "date", "relational"}
	}
	return []string{"vacation"}
}

// vacationActionBlock generates the vacation action without a require statement,
// suitable for injection into an existing script.
func vacationActionBlock(subject, body, start, end string, addresses []string) string {
	var sb strings.Builder

	useDates := start != "" && end != ""

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
