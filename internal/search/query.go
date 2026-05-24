// Package search parses the letrvu advanced search query syntax into a
// structured Query that both the IMAP layer and the local index can consume.
//
// Syntax:
//
//	from:alice          sender contains "alice"
//	to:bob              any recipient field (To/CC/BCC) contains "bob"
//	subject:invoice     subject contains "invoice"
//	has:attachment      has one or more file attachments
//	is:unread           only unread messages
//	is:read             only read messages
//	is:flagged          only flagged/starred messages
//	before:2024-01-01   sent before the given date (YYYY-MM-DD)
//	after:2023-06-01    sent on or after the given date (YYYY-MM-DD)
//	hello world         bare words matched against all text (TEXT search)
//
// Multiple tokens are ANDed together. Quoted values are supported:
//
//	from:"alice smith"  subject:"Q4 report"
package search

import (
	"strings"
	"time"
	"unicode"
)

// Query holds the parsed representation of a search expression.
type Query struct {
	// Text is a list of bare-word terms that should match anywhere in the
	// message (subject, body, headers). Each element is ANDed.
	Text []string

	// Header-specific filters.
	From    string
	To      string
	Subject string

	// Structural filters.
	HasAttachment bool

	// Status filters. nil means "no filter".
	IsRead    *bool
	IsFlagged *bool

	// Date filters (zero value = not set).
	Before time.Time
	After  time.Time
}

// IsEmpty reports whether q contains no constraints at all.
func (q *Query) IsEmpty() bool {
	return len(q.Text) == 0 &&
		q.From == "" && q.To == "" && q.Subject == "" &&
		!q.HasAttachment && q.IsRead == nil && q.IsFlagged == nil &&
		q.Before.IsZero() && q.After.IsZero()
}

// Parse tokenises raw and returns a Query. It never returns an error;
// unrecognised key:value pairs are treated as bare text.
func Parse(raw string) Query {
	var q Query
	for _, tok := range tokenise(raw) {
		key, val, ok := strings.Cut(tok, ":")
		if !ok {
			q.Text = append(q.Text, tok)
			continue
		}
		key = strings.ToLower(strings.TrimSpace(key))
		val = strings.TrimSpace(val)
		switch key {
		case "from":
			q.From = val
		case "to", "cc", "bcc":
			// Map cc/bcc onto the same "to" filter — keeps the UI simple and
			// the IMAP layer will search the correct header field.
			if q.To == "" {
				q.To = val
			} else {
				q.To += " " + val
			}
		case "subject":
			q.Subject = val
		case "has":
			if strings.ToLower(val) == "attachment" {
				q.HasAttachment = true
			} else {
				q.Text = append(q.Text, tok) // unknown, treat as text
			}
		case "is":
			switch strings.ToLower(val) {
			case "unread":
				f := false
				q.IsRead = &f
			case "read":
				t := true
				q.IsRead = &t
			case "flagged", "starred":
				t := true
				q.IsFlagged = &t
			case "unflagged", "unstarred":
				f := false
				q.IsFlagged = &f
			default:
				q.Text = append(q.Text, tok)
			}
		case "before":
			if t, err := parseDate(val); err == nil {
				q.Before = t
			} else {
				q.Text = append(q.Text, tok)
			}
		case "after":
			if t, err := parseDate(val); err == nil {
				q.After = t
			} else {
				q.Text = append(q.Text, tok)
			}
		default:
			q.Text = append(q.Text, tok)
		}
	}
	return q
}

// tokenise splits raw into tokens, respecting double-quoted strings so that
// values like from:"alice smith" are kept together.
func tokenise(raw string) []string {
	var tokens []string
	var cur strings.Builder
	inQuote := false
	for _, r := range raw {
		switch {
		case r == '"':
			inQuote = !inQuote
		case unicode.IsSpace(r) && !inQuote:
			if cur.Len() > 0 {
				tokens = append(tokens, cur.String())
				cur.Reset()
			}
		default:
			cur.WriteRune(r)
		}
	}
	if cur.Len() > 0 {
		tokens = append(tokens, cur.String())
	}
	return tokens
}

// parseDate accepts YYYY-MM-DD and returns the UTC midnight of that day.
func parseDate(s string) (time.Time, error) {
	return time.Parse("2006-01-02", s)
}

// Accessor methods so Query satisfies the imap.searchQuery interface without
// creating an import cycle.

func (q Query) GetText() []string   { return q.Text }
func (q Query) GetFrom() string     { return q.From }
func (q Query) GetTo() string       { return q.To }
func (q Query) GetSubject() string  { return q.Subject }
func (q Query) GetHasAttachment() bool { return q.HasAttachment }
func (q Query) GetIsRead() *bool    { return q.IsRead }
func (q Query) GetIsFlagged() *bool { return q.IsFlagged }
func (q Query) GetBefore() time.Time { return q.Before }
func (q Query) GetAfter() time.Time  { return q.After }
