package contacts

import (
	"strings"
	"testing"
)

// --- FormatVCard / ParseVCards round-trip ------------------------------------

func TestFormatParseVCard_RoundTrip(t *testing.T) {
	original := Contact{
		Name:  "Alice Smith",
		Notes: "met at conference",
		Emails: []ContactEmail{
			{Email: "alice@example.com", Label: "work"},
			{Email: "alice.home@example.com", Label: "home"},
		},
	}

	vcf := FormatVCard(original)
	parsed, err := ParseVCards(strings.NewReader(vcf))
	if err != nil {
		t.Fatalf("ParseVCards: %v", err)
	}
	if len(parsed) != 1 {
		t.Fatalf("want 1 contact, got %d", len(parsed))
	}
	got := parsed[0]

	if got.Name != original.Name {
		t.Errorf("Name = %q, want %q", got.Name, original.Name)
	}
	if got.Notes != original.Notes {
		t.Errorf("Notes = %q, want %q", got.Notes, original.Notes)
	}
	if len(got.Emails) != len(original.Emails) {
		t.Fatalf("len(Emails) = %d, want %d", len(got.Emails), len(original.Emails))
	}
	for i, want := range original.Emails {
		if got.Emails[i].Email != want.Email {
			t.Errorf("Emails[%d].Email = %q, want %q", i, got.Emails[i].Email, want.Email)
		}
		if got.Emails[i].Label != want.Label {
			t.Errorf("Emails[%d].Label = %q, want %q", i, got.Emails[i].Label, want.Label)
		}
	}
}

func TestFormatVCard_NoNotes(t *testing.T) {
	c := Contact{Name: "Bob", Emails: []ContactEmail{{Email: "bob@example.com"}}}
	vcf := FormatVCard(c)
	if strings.Contains(vcf, "NOTE") {
		t.Error("vCard should not contain NOTE when notes is empty")
	}
}

func TestFormatVCard_NoEmailLabel(t *testing.T) {
	c := Contact{
		Name:   "Carol",
		Emails: []ContactEmail{{Email: "carol@example.com", Label: ""}},
	}
	vcf := FormatVCard(c)
	parsed, _ := ParseVCards(strings.NewReader(vcf))
	if len(parsed) == 0 {
		t.Fatal("expected 1 contact")
	}
	if parsed[0].Emails[0].Email != "carol@example.com" {
		t.Errorf("Email = %q", parsed[0].Emails[0].Email)
	}
}

func TestFormatVCards_Multiple(t *testing.T) {
	list := []Contact{
		{Name: "Alice", Emails: []ContactEmail{{Email: "a@example.com"}}},
		{Name: "Bob", Emails: []ContactEmail{{Email: "b@example.com"}}},
	}
	vcf := FormatVCards(list)
	parsed, err := ParseVCards(strings.NewReader(vcf))
	if err != nil {
		t.Fatalf("ParseVCards: %v", err)
	}
	if len(parsed) != 2 {
		t.Fatalf("want 2 contacts, got %d", len(parsed))
	}
}

func TestParseVCards_SkipsEmpty(t *testing.T) {
	// A vCard with neither name nor emails should be ignored.
	vcf := "BEGIN:VCARD\r\nVERSION:3.0\r\nNOTE:just a note\r\nEND:VCARD\r\n"
	parsed, err := ParseVCards(strings.NewReader(vcf))
	if err != nil {
		t.Fatalf("ParseVCards: %v", err)
	}
	if len(parsed) != 0 {
		t.Errorf("want 0 contacts for name-less/email-less vCard, got %d", len(parsed))
	}
}

func TestParseVCards_MultipleEmails(t *testing.T) {
	vcf := "BEGIN:VCARD\r\nVERSION:3.0\r\nFN:Dave\r\n" +
		"EMAIL;TYPE=work:dave@work.com\r\n" +
		"EMAIL;TYPE=home:dave@home.com\r\n" +
		"END:VCARD\r\n"
	parsed, err := ParseVCards(strings.NewReader(vcf))
	if err != nil {
		t.Fatalf("ParseVCards: %v", err)
	}
	if len(parsed) != 1 {
		t.Fatalf("want 1 contact, got %d", len(parsed))
	}
	if len(parsed[0].Emails) != 2 {
		t.Errorf("want 2 emails, got %d", len(parsed[0].Emails))
	}
}

func TestParseVCards_InvalidInput(t *testing.T) {
	// go-vcard is lenient and won't error on arbitrary text — it simply
	// returns no contacts since there are no valid BEGIN:VCARD blocks.
	parsed, err := ParseVCards(strings.NewReader("not a vcard at all"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(parsed) != 0 {
		t.Errorf("want 0 contacts for non-vCard input, got %d", len(parsed))
	}
}

func TestParseVCards_EmptyInput(t *testing.T) {
	// Empty reader should return no contacts without error.
	parsed, err := ParseVCards(strings.NewReader(""))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(parsed) != 0 {
		t.Errorf("want 0 contacts for empty input, got %d", len(parsed))
	}
}

func TestFormatVCard_ContainsRequiredFields(t *testing.T) {
	c := Contact{Name: "Eve", Emails: []ContactEmail{{Email: "eve@example.com"}}}
	vcf := FormatVCard(c)
	for _, want := range []string{"BEGIN:VCARD", "END:VCARD", "FN:Eve", "VERSION:3.0"} {
		if !strings.Contains(vcf, want) {
			t.Errorf("vCard missing %q", want)
		}
	}
}
