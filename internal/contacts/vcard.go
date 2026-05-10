package contacts

import (
	"bytes"
	"fmt"
	"io"
	"strings"

	"github.com/emersion/go-vcard"
)

// FormatVCard serialises a single Contact to a vCard 3.0 string.
func FormatVCard(c Contact) string {
	card := vcard.Card{}
	card.SetValue(vcard.FieldVersion, "3.0")
	card.SetValue(vcard.FieldFormattedName, c.Name)
	if c.Notes != "" {
		card.SetValue(vcard.FieldNote, c.Notes)
	}
	for _, e := range c.Emails {
		f := &vcard.Field{Value: e.Email}
		if e.Label != "" {
			f.Params = vcard.Params{vcard.ParamType: []string{e.Label}}
		}
		card[vcard.FieldEmail] = append(card[vcard.FieldEmail], f)
	}
	var buf bytes.Buffer
	enc := vcard.NewEncoder(&buf)
	enc.Encode(card) //nolint:errcheck
	return buf.String()
}

// FormatVCards serialises a slice of contacts to a single vCard file.
func FormatVCards(contacts []Contact) string {
	var sb strings.Builder
	for _, c := range contacts {
		sb.WriteString(FormatVCard(c))
	}
	return sb.String()
}

// ParseVCards reads a vCard file and returns a slice of Contact values.
// Only the FN, NOTE, and EMAIL fields are extracted.
func ParseVCards(r io.Reader) ([]Contact, error) {
	dec := vcard.NewDecoder(r)
	var contacts []Contact
	for {
		card, err := dec.Decode()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("parse vcard: %w", err)
		}
		c := Contact{
			Name:  fieldValue(card, vcard.FieldFormattedName),
			Notes: fieldValue(card, vcard.FieldNote),
		}
		for _, f := range card[vcard.FieldEmail] {
			label := ""
			if types := f.Params[vcard.ParamType]; len(types) > 0 {
				label = types[0]
			}
			c.Emails = append(c.Emails, ContactEmail{
				Email: strings.TrimSpace(f.Value),
				Label: label,
			})
		}
		if c.Name != "" || len(c.Emails) > 0 {
			contacts = append(contacts, c)
		}
	}
	return contacts, nil
}

func fieldValue(card vcard.Card, field string) string {
	if f := card.Get(field); f != nil {
		return f.Value
	}
	return ""
}
