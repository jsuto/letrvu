package api

import "testing"

func TestParseUnsubscribeURLs_HTTPS(t *testing.T) {
	https, mailto := parseUnsubscribeURLs("<https://example.com/unsub?token=abc>")
	if https != "https://example.com/unsub?token=abc" {
		t.Errorf("httpsURL: got %q", https)
	}
	if mailto != "" {
		t.Errorf("mailtoURI: expected empty, got %q", mailto)
	}
}

func TestParseUnsubscribeURLs_Mailto(t *testing.T) {
	https, mailto := parseUnsubscribeURLs("<mailto:unsub@example.com?subject=Unsubscribe>")
	if https != "" {
		t.Errorf("httpsURL: expected empty, got %q", https)
	}
	if mailto != "mailto:unsub@example.com?subject=Unsubscribe" {
		t.Errorf("mailtoURI: got %q", mailto)
	}
}

func TestParseUnsubscribeURLs_Both(t *testing.T) {
	header := "<mailto:unsub@example.com?subject=Unsubscribe>, <https://example.com/unsub>"
	https, mailto := parseUnsubscribeURLs(header)
	if https != "https://example.com/unsub" {
		t.Errorf("httpsURL: got %q", https)
	}
	if mailto != "mailto:unsub@example.com?subject=Unsubscribe" {
		t.Errorf("mailtoURI: got %q", mailto)
	}
}

func TestParseUnsubscribeURLs_HTTPSkipped(t *testing.T) {
	// Plain http:// URLs should not be returned (we only accept https://).
	https, mailto := parseUnsubscribeURLs("<http://example.com/unsub>")
	if https != "" {
		t.Errorf("plain http should not be returned, got %q", https)
	}
	if mailto != "" {
		t.Errorf("mailto: expected empty, got %q", mailto)
	}
}

func TestParseUnsubscribeURLs_Empty(t *testing.T) {
	https, mailto := parseUnsubscribeURLs("")
	if https != "" || mailto != "" {
		t.Errorf("expected both empty, got https=%q mailto=%q", https, mailto)
	}
}

func TestParseMailtoURI_Full(t *testing.T) {
	addr, subject, body := parseMailtoURI("mailto:unsub@example.com?subject=Unsubscribe&body=Please+remove+me")
	if addr != "unsub@example.com" {
		t.Errorf("addr: got %q", addr)
	}
	if subject != "Unsubscribe" {
		t.Errorf("subject: got %q", subject)
	}
	if body != "Please remove me" {
		t.Errorf("body: got %q", body)
	}
}

func TestParseMailtoURI_NoQuery(t *testing.T) {
	addr, subject, body := parseMailtoURI("mailto:unsub@example.com")
	if addr != "unsub@example.com" {
		t.Errorf("addr: got %q", addr)
	}
	if subject != "" || body != "" {
		t.Errorf("expected empty subject/body, got %q %q", subject, body)
	}
}

func TestParseMailtoURI_Invalid(t *testing.T) {
	addr, _, _ := parseMailtoURI("https://not-a-mailto.com")
	if addr != "" {
		t.Errorf("expected empty addr for non-mailto URI, got %q", addr)
	}
}
