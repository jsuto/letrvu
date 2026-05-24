package search

import (
	"testing"
	"time"
)

func TestParse_BareWords(t *testing.T) {
	q := Parse("hello world")
	if len(q.Text) != 2 || q.Text[0] != "hello" || q.Text[1] != "world" {
		t.Errorf("unexpected text terms: %v", q.Text)
	}
}

func TestParse_FromFilter(t *testing.T) {
	q := Parse("from:alice")
	if q.From != "alice" {
		t.Errorf("want from=alice, got %q", q.From)
	}
	if len(q.Text) != 0 {
		t.Errorf("unexpected text terms: %v", q.Text)
	}
}

func TestParse_FromQuoted(t *testing.T) {
	q := Parse(`from:"alice smith"`)
	if q.From != "alice smith" {
		t.Errorf("want from=%q, got %q", "alice smith", q.From)
	}
}

func TestParse_SubjectFilter(t *testing.T) {
	q := Parse("subject:invoice")
	if q.Subject != "invoice" {
		t.Errorf("want subject=invoice, got %q", q.Subject)
	}
}

func TestParse_ToFilter(t *testing.T) {
	q := Parse("to:bob")
	if q.To != "bob" {
		t.Errorf("want to=bob, got %q", q.To)
	}
}

func TestParse_CcMapsTTo(t *testing.T) {
	q := Parse("cc:carol")
	if q.To != "carol" {
		t.Errorf("cc: should map to To field, got %q", q.To)
	}
}

func TestParse_HasAttachment(t *testing.T) {
	q := Parse("has:attachment")
	if !q.HasAttachment {
		t.Error("want HasAttachment=true")
	}
}

func TestParse_IsUnread(t *testing.T) {
	q := Parse("is:unread")
	if q.IsRead == nil || *q.IsRead != false {
		t.Error("is:unread should set IsRead=&false")
	}
}

func TestParse_IsRead(t *testing.T) {
	q := Parse("is:read")
	if q.IsRead == nil || *q.IsRead != true {
		t.Error("is:read should set IsRead=&true")
	}
}

func TestParse_IsFlagged(t *testing.T) {
	q := Parse("is:flagged")
	if q.IsFlagged == nil || *q.IsFlagged != true {
		t.Error("is:flagged should set IsFlagged=&true")
	}
}

func TestParse_IsStarred(t *testing.T) {
	q := Parse("is:starred")
	if q.IsFlagged == nil || *q.IsFlagged != true {
		t.Error("is:starred should set IsFlagged=&true")
	}
}

func TestParse_BeforeDate(t *testing.T) {
	q := Parse("before:2024-01-01")
	want := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	if !q.Before.Equal(want) {
		t.Errorf("want before=%v, got %v", want, q.Before)
	}
}

func TestParse_AfterDate(t *testing.T) {
	q := Parse("after:2023-06-01")
	want := time.Date(2023, 6, 1, 0, 0, 0, 0, time.UTC)
	if !q.After.Equal(want) {
		t.Errorf("want after=%v, got %v", want, q.After)
	}
}

func TestParse_InvalidDateFallsBackToText(t *testing.T) {
	q := Parse("before:notadate")
	if len(q.Text) != 1 || q.Text[0] != "before:notadate" {
		t.Errorf("invalid date should fall back to text, got text=%v", q.Text)
	}
	if !q.Before.IsZero() {
		t.Error("Before should remain zero")
	}
}

func TestParse_UnknownKeyFallsBackToText(t *testing.T) {
	q := Parse("foobar:xyz")
	if len(q.Text) != 1 || q.Text[0] != "foobar:xyz" {
		t.Errorf("unknown key:val should be text, got %v", q.Text)
	}
}

func TestParse_Mixed(t *testing.T) {
	q := Parse("from:alice subject:invoice is:unread")
	if q.From != "alice" {
		t.Errorf("from: %q", q.From)
	}
	if q.Subject != "invoice" {
		t.Errorf("subject: %q", q.Subject)
	}
	if q.IsRead == nil || *q.IsRead != false {
		t.Error("is:unread")
	}
	if len(q.Text) != 0 {
		t.Errorf("no bare text expected, got %v", q.Text)
	}
}

func TestIsEmpty(t *testing.T) {
	q := Parse("")
	if !q.IsEmpty() {
		t.Error("empty string should produce empty query")
	}
	q2 := Parse("hello")
	if q2.IsEmpty() {
		t.Error("non-empty query should not be empty")
	}
}
