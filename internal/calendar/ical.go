package calendar

import (
	"bytes"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/emersion/go-ical"
)

// ParseICal parses an iCalendar string and returns a slice of Events.
func ParseICal(src string) ([]Event, error) {
	dec := ical.NewDecoder(strings.NewReader(src))
	cal, err := dec.Decode()
	if err != nil {
		return nil, fmt.Errorf("parse ical: %w", err)
	}

	var events []Event
	for _, comp := range cal.Children {
		if comp.Name != ical.CompEvent {
			continue
		}
		ev, err := componentToEvent(comp)
		if err != nil {
			continue // skip malformed events
		}
		events = append(events, ev)
	}
	return events, nil
}

// FormatICal serialises a slice of Events to an iCalendar string.
func FormatICal(events []Event) string {
	cal := ical.NewCalendar()
	cal.Props.SetText(ical.PropVersion, "2.0")
	cal.Props.SetText(ical.PropProductID, "-//letrvu//letrvu//EN")

	for _, ev := range events {
		comp := eventToComponent(ev)
		cal.Children = append(cal.Children, comp)
	}

	var buf bytes.Buffer
	enc := ical.NewEncoder(&buf)
	enc.Encode(cal) //nolint:errcheck
	return buf.String()
}

// FormatICalSingle serialises a single Event to an iCalendar string.
func FormatICalSingle(ev Event) string {
	return FormatICal([]Event{ev})
}

// FormatICalInvite serialises a single Event as a meeting invite
// (METHOD:REQUEST) with ORGANIZER and ATTENDEE fields set. This produces an
// iCalendar payload that mail clients interpret as an accept/decline invitation.
//
// organizer is the sender's email address ("alice@example.com").
// attendees is the list of recipient addresses. Duplicates and the organizer
// address are silently deduplicated. An empty attendees slice is valid — the
// resulting invite still carries METHOD:REQUEST and the ORGANIZER field, but
// mail clients may not show accept/decline buttons without at least one ATTENDEE.
func FormatICalInvite(ev Event, organizer string, attendees []string) string {
	cal := ical.NewCalendar()
	cal.Props.SetText(ical.PropVersion, "2.0")
	cal.Props.SetText(ical.PropProductID, "-//letrvu//letrvu//EN")
	cal.Props.SetText("METHOD", "REQUEST")

	comp := eventToComponent(ev)
	comp.Props.SetText("STATUS", "CONFIRMED")

	// ORGANIZER
	orgProp := ical.Prop{
		Name:   "ORGANIZER",
		Value:  "mailto:" + organizer,
		Params: make(ical.Params),
	}
	orgProp.Params.Set("CN", organizer)
	comp.Props["ORGANIZER"] = []ical.Prop{orgProp}

	// ATTENDEE — one entry per unique address; skip the organizer's own address
	seen := map[string]bool{strings.ToLower(organizer): true}
	for _, addr := range attendees {
		lc := strings.ToLower(strings.TrimSpace(addr))
		if lc == "" || seen[lc] {
			continue
		}
		seen[lc] = true
		att := ical.Prop{
			Name:   "ATTENDEE",
			Value:  "mailto:" + addr,
			Params: make(ical.Params),
		}
		att.Params.Set("CUTYPE", "INDIVIDUAL")
		att.Params.Set("ROLE", "REQ-PARTICIPANT")
		att.Params.Set("PARTSTAT", "NEEDS-ACTION")
		att.Params.Set("RSVP", "TRUE")
		comp.Props["ATTENDEE"] = append(comp.Props["ATTENDEE"], att)
	}

	cal.Children = append(cal.Children, comp)

	var buf bytes.Buffer
	enc := ical.NewEncoder(&buf)
	enc.Encode(cal) //nolint:errcheck
	return buf.String()
}

func componentToEvent(comp *ical.Component) (Event, error) {
	var ev Event

	if p := comp.Props.Get(ical.PropSummary); p != nil {
		ev.Title = p.Value
	}
	if p := comp.Props.Get(ical.PropDescription); p != nil {
		ev.Description = p.Value
	}
	if p := comp.Props.Get(ical.PropLocation); p != nil {
		ev.Location = p.Value
	}
	if p := comp.Props.Get(ical.PropRecurrenceRule); p != nil {
		ev.Rrule = p.Value
	}

	dtstart := comp.Props.Get(ical.PropDateTimeStart)
	dtend := comp.Props.Get(ical.PropDateTimeEnd)

	if dtstart == nil {
		return ev, fmt.Errorf("missing DTSTART")
	}

	// Detect all-day events (DATE value type, no time component).
	valueType := dtstart.Params.Get(ical.ParamValue)
	if valueType == "DATE" {
		ev.AllDay = true
		ev.StartsAt, _ = time.Parse("20060102", dtstart.Value)
		if dtend != nil {
			ev.EndsAt, _ = time.Parse("20060102", dtend.Value)
			ev.EndsAt = ev.EndsAt.Add(-24 * time.Hour) // iCal end is exclusive
		} else {
			ev.EndsAt = ev.StartsAt
		}
	} else {
		ev.StartsAt = parseICalTime(dtstart.Value)
		if dtend != nil {
			ev.EndsAt = parseICalTime(dtend.Value)
		} else {
			ev.EndsAt = ev.StartsAt.Add(time.Hour)
		}
	}

	return ev, nil
}

func eventToComponent(ev Event) *ical.Component {
	comp := ical.NewComponent(ical.CompEvent)

	uid := fmt.Sprintf("letrvu-%d@letrvu", ev.ID)
	comp.Props.SetText(ical.PropUID, uid)
	comp.Props.SetText(ical.PropSummary, ev.Title)
	if ev.Description != "" {
		comp.Props.SetText(ical.PropDescription, ev.Description)
	}
	if ev.Location != "" {
		comp.Props.SetText(ical.PropLocation, ev.Location)
	}

	if ev.AllDay {
		start := ical.Prop{Name: ical.PropDateTimeStart, Value: ev.StartsAt.Format("20060102"), Params: make(ical.Params)}
		start.Params.Set(ical.ParamValue, "DATE")
		end := ical.Prop{Name: ical.PropDateTimeEnd, Value: ev.EndsAt.Add(24 * time.Hour).Format("20060102"), Params: make(ical.Params)}
		end.Params.Set(ical.ParamValue, "DATE")
		comp.Props[ical.PropDateTimeStart] = []ical.Prop{start}
		comp.Props[ical.PropDateTimeEnd] = []ical.Prop{end}
	} else {
		comp.Props.SetDateTime(ical.PropDateTimeStart, ev.StartsAt.UTC())
		comp.Props.SetDateTime(ical.PropDateTimeEnd, ev.EndsAt.UTC())
	}

	now := ical.Prop{Name: ical.PropDateTimeStamp, Value: time.Now().UTC().Format("20060102T150405Z")}
	comp.Props[ical.PropDateTimeStamp] = []ical.Prop{now}

	if ev.Rrule != "" {
		rruleProp := ical.Prop{Name: ical.PropRecurrenceRule, Value: ev.Rrule, Params: make(ical.Params)}
		rruleProp.SetValueType("RECUR")
		comp.Props[ical.PropRecurrenceRule] = []ical.Prop{rruleProp}
	}

	return comp
}

func parseICalTime(s string) time.Time {
	formats := []string{
		"20060102T150405Z",
		"20060102T150405",
		"20060102",
	}
	for _, f := range formats {
		if t, err := time.Parse(f, s); err == nil {
			return t
		}
	}
	return time.Time{}
}

// ReadAll reads all bytes from r into a string (convenience for parsing).
func ReadAll(r io.Reader) (string, error) {
	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		return "", err
	}
	return buf.String(), nil
}
