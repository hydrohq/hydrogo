package bffproxy

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

type sessionStub struct {
	cookie string
	err    error
}

func (s sessionStub) CoreCookie(context.Context, *http.Request) (string, error) {
	return s.cookie, s.err
}

type csrfStub struct{ err error }

func (s csrfStub) Validate(context.Context, *http.Request) error { return s.err }

func TestHandlerUsesServerSessionAndStripsBrowserBoundary(t *testing.T) {
	core := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Cookie"); got != "hydro_session=server" {
			t.Fatalf("core cookie = %q", got)
		}
		if got := r.Header.Get("Authorization"); got != "" {
			t.Fatalf("authorization leaked: %q", got)
		}
		if got := r.Header.Get("Origin"); got != "" {
			t.Fatalf("origin leaked: %q", got)
		}
		w.Header().Set("Set-Cookie", "hydro_session=core-secret; HttpOnly")
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"ok":true}`)
	}))
	defer core.Close()
	origin, err := url.Parse(core.URL)
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodGet, "http://bff.local/api/v1/pages", nil)
	request.Header.Set("Cookie", "bff_session=browser")
	request.Header.Set("Origin", "https://frontend.example")
	recorder := httptest.NewRecorder()
	Handler{CoreOrigin: origin, Sessions: sessionStub{cookie: "hydro_session=server"}}.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK || recorder.Body.String() != `{"ok":true}` {
		t.Fatalf("response = %d %s", recorder.Code, recorder.Body.String())
	}
	if recorder.Header().Get("Set-Cookie") != "" {
		t.Fatal("core Set-Cookie leaked to browser")
	}
}

func TestHandlerRequiresCSRFForMutations(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "http://bff.local/api/v1/pages", nil)
	recorder := httptest.NewRecorder()
	Handler{Sessions: sessionStub{cookie: "server"}}.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusInternalServerError)
	}

	recorder = httptest.NewRecorder()
	Handler{Sessions: sessionStub{cookie: "server"}, CSRF: csrfStub{err: errors.New("bad")}}.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusForbidden)
	}
}

func TestHandlerBoundsResponse(t *testing.T) {
	core := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, _ = io.WriteString(w, "12345") }))
	defer core.Close()
	origin, _ := url.Parse(core.URL)
	request := httptest.NewRequest(http.MethodGet, "http://bff.local/api/v1/pages", nil)
	recorder := httptest.NewRecorder()
	Handler{CoreOrigin: origin, Sessions: sessionStub{cookie: "server"}, MaxResponseBytes: 4}.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusBadGateway {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusBadGateway)
	}
}

func TestHandlerBoundsRequest(t *testing.T) {
	core := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.Copy(io.Discard, r.Body)
	}))
	defer core.Close()
	origin, _ := url.Parse(core.URL)
	request := httptest.NewRequest(http.MethodPost, "http://bff.local/api/v1/pages", bytes.NewReader([]byte("12345")))
	recorder := httptest.NewRecorder()
	Handler{CoreOrigin: origin, Sessions: sessionStub{cookie: "server"}, CSRF: csrfStub{}, MaxRequestBytes: 4}.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusRequestEntityTooLarge)
	}
}
