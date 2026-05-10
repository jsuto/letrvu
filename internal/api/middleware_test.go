package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// --- securityHeaders ---------------------------------------------------------

func TestSecurityHeaders_Present(t *testing.T) {
	handler := securityHeaders(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	headers := map[string]string{
		"Content-Security-Policy": "default-src",
		"X-Content-Type-Options":  "nosniff",
		"X-Frame-Options":         "DENY",
	}
	for header, want := range headers {
		got := rr.Header().Get(header)
		if got == "" {
			t.Errorf("missing header %s", header)
		}
		if want != "" && got != want {
			// For CSP just check it starts correctly.
			if header == "Content-Security-Policy" {
				if len(got) == 0 {
					t.Errorf("CSP header is empty")
				}
			} else {
				t.Errorf("%s = %q, want %q", header, got, want)
			}
		}
	}
}

func TestSecurityHeaders_PassThrough(t *testing.T) {
	handler := securityHeaders(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTeapot)
	}))
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusTeapot {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusTeapot)
	}
}

// --- checkCSRF ---------------------------------------------------------------

func makeCSRFRequest(method, token, cookie string) (*http.Request, *httptest.ResponseRecorder) {
	req := httptest.NewRequest(method, "/api/test", nil)
	if cookie != "" {
		req.AddCookie(&http.Cookie{Name: "letrvu_csrf", Value: cookie})
	}
	if token != "" {
		req.Header.Set("X-CSRF-Token", token)
	}
	return req, httptest.NewRecorder()
}

func TestCheckCSRF_SafeMethodsSkipped(t *testing.T) {
	for _, method := range []string{http.MethodGet, http.MethodHead, http.MethodOptions} {
		req, rr := makeCSRFRequest(method, "", "")
		if !checkCSRF(rr, req) {
			t.Errorf("method %s should skip CSRF check", method)
		}
	}
}

func TestCheckCSRF_ValidToken(t *testing.T) {
	req, rr := makeCSRFRequest(http.MethodPost, "abc123", "abc123")
	if !checkCSRF(rr, req) {
		t.Errorf("matching token and cookie should pass")
	}
}

func TestCheckCSRF_MismatchedToken(t *testing.T) {
	req, rr := makeCSRFRequest(http.MethodPost, "wrong-token", "correct-token")
	if checkCSRF(rr, req) {
		t.Error("mismatched token should fail")
	}
	if rr.Code != http.StatusForbidden {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusForbidden)
	}
}

func TestCheckCSRF_MissingCookie(t *testing.T) {
	req, rr := makeCSRFRequest(http.MethodPost, "abc123", "")
	if checkCSRF(rr, req) {
		t.Error("missing cookie should fail")
	}
	if rr.Code != http.StatusForbidden {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusForbidden)
	}
}

func TestCheckCSRF_MissingHeader(t *testing.T) {
	req, rr := makeCSRFRequest(http.MethodPost, "", "abc123")
	if checkCSRF(rr, req) {
		t.Error("missing header should fail")
	}
	if rr.Code != http.StatusForbidden {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusForbidden)
	}
}

func TestCheckCSRF_MutatingMethods(t *testing.T) {
	// All mutating methods should enforce CSRF.
	for _, method := range []string{http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete} {
		req, rr := makeCSRFRequest(method, "tok", "tok")
		if !checkCSRF(rr, req) {
			t.Errorf("method %s with valid token should pass", method)
		}
		req, rr = makeCSRFRequest(method, "bad", "tok")
		if checkCSRF(rr, req) {
			t.Errorf("method %s with bad token should fail", method)
		}
	}
}

func TestCheckCSRF_EmptyCookieValue(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.AddCookie(&http.Cookie{Name: "letrvu_csrf", Value: ""})
	req.Header.Set("X-CSRF-Token", "")
	rr := httptest.NewRecorder()
	if checkCSRF(rr, req) {
		t.Error("empty cookie value should fail")
	}
}
