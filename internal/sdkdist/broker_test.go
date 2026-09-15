package sdkdist

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestRequestGrantParsesBoundedResponseWithoutPersistingToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.Header.Get("Authorization") != "Bearer secret-token" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.Header.Get("Authorization"))
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"artifact","project_id":"p","core_version":"c","sdk_version":"s","artifact_url":"http://127.0.0.1/artifact","issued_at":"2026-09-16T12:00:00Z","expires_at":"2026-09-16T12:05:00Z"}`))
	}))
	defer server.Close()
	grant, err := RequestGrant(context.Background(), server.URL, "secret-token", GrantRequest{ProjectID: "p", CoreVersion: "c", SDKVersion: "s", Target: "linux-amd64", Edition: "growth"}, nil, true)
	if err != nil || grant.ProjectID != "p" || grant.AccessToken != "artifact" {
		t.Fatalf("grant=%#v err=%v", grant, err)
	}
}

func TestRequestGrantRejectsInvalidCoordinatesBeforeNetwork(t *testing.T) {
	called := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { called = true }))
	defer server.Close()
	if _, err := RequestGrant(context.Background(), server.URL, "token", GrantRequest{ProjectID: "p"}, nil, true); err == nil {
		t.Fatal("invalid grant coordinates accepted")
	}
	if called {
		t.Fatal("invalid grant request reached broker")
	}
}

func TestRequestValidatedGrantRejectsLockMismatch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"access_token":"artifact","project_id":"p","core_version":"c","sdk_version":"s","artifact_url":"http://127.0.0.1/artifact","issued_at":"2026-09-16T12:00:00Z","expires_at":"2026-09-16T12:05:00Z"}`))
	}))
	defer server.Close()
	request := GrantRequest{ProjectID: "p", CoreVersion: "c", SDKVersion: "s", Target: "linux-amd64", Edition: "growth"}
	grant, err := RequestValidatedGrant(context.Background(), server.URL, "token", request, "wrong", "c", "s", time.Date(2026, 9, 16, 12, 1, 0, 0, time.UTC), nil, true)
	if err == nil || grant.AccessToken != "" || !strings.Contains(err.Error(), "project") {
		t.Fatalf("grant=%#v err=%v", grant, err)
	}
}

func TestRequestGrantRejectsOversizedResponseAndErrors(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(strings.Repeat("x", MaxGrantResponseBytes+1)))
	}))
	defer server.Close()
	validRequest := GrantRequest{ProjectID: "p", CoreVersion: "c", SDKVersion: "s", Target: "linux-amd64", Edition: "growth"}
	if _, err := RequestGrant(context.Background(), server.URL, "token", validRequest, nil, true); err == nil {
		t.Fatal("oversized broker response accepted")
	}
	status := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { http.Error(w, "no", http.StatusForbidden) }))
	defer status.Close()
	if _, err := RequestGrant(context.Background(), status.URL, "token", validRequest, nil, true); err == nil || !strings.Contains(err.Error(), "HTTP 403") {
		t.Fatalf("status error=%v", err)
	}
}
