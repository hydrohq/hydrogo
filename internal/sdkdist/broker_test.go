package sdkdist

import (
	"context"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/testhttp"
)

func TestRequestGrantParsesBoundedResponseWithoutPersistingToken(t *testing.T) {
	const endpoint = "http://127.0.0.1:19101"
	client := testhttp.Client(testhttp.Handlers{"127.0.0.1:19101": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.Header.Get("Authorization") != "Bearer secret-token" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.Header.Get("Authorization"))
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"artifact","project_id":"p","core_version":"c","sdk_version":"s","artifact_url":"http://127.0.0.1/artifact","issued_at":"2026-09-16T12:00:00Z","expires_at":"2026-09-16T12:05:00Z"}`))
	})})
	grant, err := RequestGrant(context.Background(), endpoint, "secret-token", GrantRequest{ProjectID: "p", CoreVersion: "c", SDKVersion: "s", Target: "linux-amd64", Edition: "growth"}, client, true)
	if err != nil || grant.ProjectID != "p" || grant.AccessToken != "artifact" {
		t.Fatalf("grant=%#v err=%v", grant, err)
	}
}

func TestRequestGrantRejectsInvalidCoordinatesBeforeNetwork(t *testing.T) {
	called := false
	const endpoint = "http://127.0.0.1:19102"
	client := testhttp.Client(testhttp.Handlers{"127.0.0.1:19102": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { called = true })})
	if _, err := RequestGrant(context.Background(), endpoint, "token", GrantRequest{ProjectID: "p"}, client, true); err == nil {
		t.Fatal("invalid grant coordinates accepted")
	}
	if called {
		t.Fatal("invalid grant request reached broker")
	}
}

func TestRequestValidatedGrantRejectsLockMismatch(t *testing.T) {
	const endpoint = "http://127.0.0.1:19103"
	client := testhttp.Client(testhttp.Handlers{"127.0.0.1:19103": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"access_token":"artifact","project_id":"p","core_version":"c","sdk_version":"s","artifact_url":"http://127.0.0.1/artifact","issued_at":"2026-09-16T12:00:00Z","expires_at":"2026-09-16T12:05:00Z"}`))
	})})
	request := GrantRequest{ProjectID: "p", CoreVersion: "c", SDKVersion: "s", Target: "linux-amd64", Edition: "growth"}
	grant, err := RequestValidatedGrant(context.Background(), endpoint, "token", request, "wrong", "c", "s", time.Date(2026, 9, 16, 12, 1, 0, 0, time.UTC), client, true)
	if err == nil || grant.AccessToken != "" || !strings.Contains(err.Error(), "project") {
		t.Fatalf("grant=%#v err=%v", grant, err)
	}
}

func TestRequestGrantRejectsOversizedResponseAndErrors(t *testing.T) {
	const oversizedEndpoint = "http://127.0.0.1:19104"
	client := testhttp.Client(testhttp.Handlers{"127.0.0.1:19104": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(strings.Repeat("x", MaxGrantResponseBytes+1)))
	})})
	validRequest := GrantRequest{ProjectID: "p", CoreVersion: "c", SDKVersion: "s", Target: "linux-amd64", Edition: "growth"}
	if _, err := RequestGrant(context.Background(), oversizedEndpoint, "token", validRequest, client, true); err == nil {
		t.Fatal("oversized broker response accepted")
	}
	const statusEndpoint = "http://127.0.0.1:19105"
	statusClient := testhttp.Client(testhttp.Handlers{"127.0.0.1:19105": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { http.Error(w, "no", http.StatusForbidden) })})
	if _, err := RequestGrant(context.Background(), statusEndpoint, "token", validRequest, statusClient, true); err == nil || !strings.Contains(err.Error(), "HTTP 403") {
		t.Fatalf("status error=%v", err)
	}
}
