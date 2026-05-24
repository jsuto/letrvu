package filters

import (
	"regexp"
	"strings"
)

// MatchInput holds the fields of an incoming message that filter conditions
// can test against.
type MatchInput struct {
	Subject       string
	From          string
	To            string // space-joined list of recipients
	Body          string
	HasAttachment bool
	SizeBytes     uint32
}

// Match returns true when the filter's conditions are satisfied.
// If the filter has no conditions it never matches (safe default).
func Match(f Filter, m MatchInput) bool {
	if !f.Enabled || len(f.Conditions) == 0 {
		return false
	}
	if f.MatchAll {
		for _, c := range f.Conditions {
			if !matchCond(c, m) {
				return false
			}
		}
		return true
	}
	// OR: at least one condition must match.
	for _, c := range f.Conditions {
		if matchCond(c, m) {
			return true
		}
	}
	return false
}

func matchCond(c Condition, m MatchInput) bool {
	fieldVal := fieldValue(c.Field, m)
	switch c.Op {
	case "contains":
		return strings.Contains(strings.ToLower(fieldVal), strings.ToLower(c.Value))
	case "not_contains":
		return !strings.Contains(strings.ToLower(fieldVal), strings.ToLower(c.Value))
	case "equals":
		return strings.EqualFold(fieldVal, c.Value)
	case "not_equals":
		return !strings.EqualFold(fieldVal, c.Value)
	case "matches":
		re, err := regexp.Compile("(?i)" + c.Value)
		if err != nil {
			return false
		}
		return re.MatchString(fieldVal)
	}
	return false
}

func fieldValue(field string, m MatchInput) string {
	switch field {
	case "subject":
		return m.Subject
	case "from":
		return m.From
	case "to":
		return m.To
	case "body":
		return m.Body
	case "has_attachment":
		if m.HasAttachment {
			return "true"
		}
		return "false"
	}
	return ""
}
