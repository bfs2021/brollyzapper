package api_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// BrollyZap-as0.4: a signed-out monitor probing the admin UI sends HEAD, and the
// login handler answered it 405 while every page behind a `GET /...` pattern
// answered HEAD from the GET handler. The monitor reported the app down.
//
// The handler switch is manual, so HEAD has to be named with GET explicitly;
// ServeMux only supplies the GET-answers-HEAD rule to patterns it routes.
func TestHeadLoginAnswersAsGetDoes(t *testing.T) {
	h := newHarness(t)

	head := httptest.NewRecorder()
	h.handler.ServeHTTP(head, httptest.NewRequest(http.MethodHead, "/login", nil))
	if head.Code != http.StatusOK {
		t.Fatalf("HEAD /login = %d, want 200; a monitor sending HEAD reports the app down",
			head.Code)
	}
	for _, want := range []struct {
		header   string
		contains string
	}{
		{"Cache-Control", "no-store"},
		{"X-Content-Type-Options", "nosniff"},
		{"Referrer-Policy", "no-referrer"},
		{"Content-Security-Policy", "default-src 'self'"},
		{"Content-Security-Policy", "frame-ancestors 'none'"},
	} {
		if got := head.Header().Get(want.header); !strings.Contains(got, want.contains) {
			t.Errorf("HEAD /login: %s = %q, want it to contain %q", want.header, got,
				want.contains)
		}
	}

	get := h.get(t, "/login", nil)
	if get.Code != http.StatusOK {
		t.Fatalf("GET /login = %d, want 200", get.Code)
	}
	headType, getType := head.Header().Get("Content-Type"), get.Header().Get("Content-Type")
	if headType != getType || headType == "" {
		t.Errorf("HEAD /login Content-Type = %q, GET /login = %q; HEAD describes the GET response",
			headType, getType)
	}

	// POST still reaches attemptLogin: a wrong password re-renders the form
	// with the error instead of being answered as a page fetch.
	b := h.browser()
	page := b.get(t, "/login")
	failed := b.submitLogin(t, csrfFrom(t, page.Body.String()), "not-the-password")
	if failed.Code != http.StatusOK {
		t.Fatalf("POST /login = %d, want 200 with the form re-rendered", failed.Code)
	}
	if !strings.Contains(failed.Body.String(), "That password is not correct.") {
		t.Error("POST /login did not run the login attempt")
	}

	// The methods that were refused before are still refused, with the rate
	// limiter and the refusal in front of them as before.
	for _, method := range []string{
		http.MethodPut, http.MethodDelete, http.MethodPatch, http.MethodOptions,
	} {
		rec := httptest.NewRecorder()
		h.handler.ServeHTTP(rec, httptest.NewRequest(method, "/login", nil))
		if rec.Code != http.StatusMethodNotAllowed {
			t.Errorf("%s /login = %d, want 405", method, rec.Code)
		}
	}
}

// httptest.ResponseRecorder keeps whatever the handler writes; only the HTTP
// server drops the body on HEAD. A monitor sees the server's response, so the
// no-body half runs through a real one.
func TestHeadLoginThroughARealServerHasNoBody(t *testing.T) {
	h := newHarness(t)
	srv := httptest.NewServer(h.handler)
	defer srv.Close()

	resp, err := srv.Client().Head(srv.URL + "/login")
	if err != nil {
		t.Fatalf("HEAD /login: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("HEAD /login = %d, want 200", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("reading the HEAD body: %v", err)
	}
	if len(body) != 0 {
		t.Errorf("HEAD /login returned %d bytes; the server drops the body", len(body))
	}
	if got := resp.Header.Get("Content-Security-Policy"); !strings.Contains(got, "frame-ancestors 'none'") {
		t.Errorf("HEAD /login through the server: Content-Security-Policy = %q, want the "+
			"admin group's policy", got)
	}
}
