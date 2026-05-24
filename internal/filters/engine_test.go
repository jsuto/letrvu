package filters

import "testing"

func TestMatch_AllOf(t *testing.T) {
	f := Filter{
		Enabled:  true,
		MatchAll: true,
		Conditions: []Condition{
			{Field: "subject", Op: "contains", Value: "invoice"},
			{Field: "from", Op: "contains", Value: "billing@"},
		},
		Actions: []Action{{Type: "move", Value: "Invoices"}},
	}

	yes := MatchInput{Subject: "Your Invoice #123", From: "billing@example.com"}
	if !Match(f, yes) {
		t.Error("expected match")
	}

	no := MatchInput{Subject: "Your Invoice #123", From: "other@example.com"}
	if Match(f, no) {
		t.Error("expected no match (AND fails)")
	}
}

func TestMatch_AnyOf(t *testing.T) {
	f := Filter{
		Enabled:  true,
		MatchAll: false,
		Conditions: []Condition{
			{Field: "subject", Op: "contains", Value: "urgent"},
			{Field: "subject", Op: "contains", Value: "important"},
		},
		Actions: []Action{{Type: "mark_flagged"}},
	}

	if !Match(f, MatchInput{Subject: "URGENT: action required"}) {
		t.Error("expected match on 'urgent'")
	}
	if !Match(f, MatchInput{Subject: "Important update"}) {
		t.Error("expected match on 'important'")
	}
	if Match(f, MatchInput{Subject: "Hello"}) {
		t.Error("expected no match")
	}
}

func TestMatch_Disabled(t *testing.T) {
	f := Filter{
		Enabled:    false,
		Conditions: []Condition{{Field: "subject", Op: "contains", Value: "x"}},
		Actions:    []Action{{Type: "mark_read"}},
	}
	if Match(f, MatchInput{Subject: "x"}) {
		t.Error("disabled filter should not match")
	}
}

func TestMatch_NoConditions(t *testing.T) {
	f := Filter{Enabled: true, Conditions: nil, Actions: []Action{{Type: "mark_read"}}}
	if Match(f, MatchInput{Subject: "anything"}) {
		t.Error("filter with no conditions should never match")
	}
}

func TestMatch_Equals(t *testing.T) {
	f := Filter{
		Enabled:    true,
		MatchAll:   true,
		Conditions: []Condition{{Field: "from", Op: "equals", Value: "boss@corp.com"}},
		Actions:    []Action{{Type: "mark_flagged"}},
	}
	if !Match(f, MatchInput{From: "BOSS@corp.com"}) {
		t.Error("equals should be case-insensitive")
	}
	if Match(f, MatchInput{From: "boss@corp.com.evil"}) {
		t.Error("equals should not match partial")
	}
}

func TestMatch_NotContains(t *testing.T) {
	f := Filter{
		Enabled:    true,
		MatchAll:   true,
		Conditions: []Condition{{Field: "subject", Op: "not_contains", Value: "spam"}},
		Actions:    []Action{{Type: "mark_read"}},
	}
	if !Match(f, MatchInput{Subject: "Hello world"}) {
		t.Error("expected match when value not present")
	}
	if Match(f, MatchInput{Subject: "This is spam"}) {
		t.Error("expected no match when value present")
	}
}

func TestMatch_Regex(t *testing.T) {
	f := Filter{
		Enabled:    true,
		MatchAll:   true,
		Conditions: []Condition{{Field: "subject", Op: "matches", Value: `invoice #\d+`}},
		Actions:    []Action{{Type: "move", Value: "Invoices"}},
	}
	if !Match(f, MatchInput{Subject: "Invoice #42"}) {
		t.Error("expected regex match")
	}
	if Match(f, MatchInput{Subject: "Invoice ABC"}) {
		t.Error("expected no regex match")
	}
}

func TestMatch_HasAttachment(t *testing.T) {
	f := Filter{
		Enabled:    true,
		MatchAll:   true,
		Conditions: []Condition{{Field: "has_attachment", Op: "equals", Value: "true"}},
		Actions:    []Action{{Type: "mark_flagged"}},
	}
	if !Match(f, MatchInput{HasAttachment: true}) {
		t.Error("expected match")
	}
	if Match(f, MatchInput{HasAttachment: false}) {
		t.Error("expected no match")
	}
}
