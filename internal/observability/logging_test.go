package observability

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/ardaozyildirim/hydrogo/internal/auth"
	"github.com/ardaozyildirim/hydrogo/internal/security"
)

func TestBoundedPathCapsLogField(t *testing.T) {
	path := make([]byte, maxLoggedPathBytes+20)
	for i := range path {
		path[i] = 'a'
	}
	bounded := boundedPath(string(path))
	if len(bounded) != maxLoggedPathBytes+len("…") {
		t.Fatalf("bounded path length = %d, want %d", len(bounded), maxLoggedPathBytes+len("…"))
	}
	if bounded[len(bounded)-len("…"):] != "…" {
		t.Fatalf("bounded path missing truncation marker: %q", bounded)
	}
}

func TestLoggingMiddlewareCorrelatesActorAndSessionWithoutPII(t *testing.T) {
	var logs strings.Builder
	logger := slog.New(slog.NewJSONHandler(&logs, nil))
	handler := LoggingMiddleware(logger)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	ctx := security.WithRequestID(context.Background(), "req-1")
	ctx = auth.WithUser(ctx, &auth.User{ID: "actor-1", Email: "private@example.test"})
	ctx = auth.WithSession(ctx, &auth.Session{ID: "session-1", TokenHash: "secret-token-hash"})
	ctx = security.WithJobID(ctx, "job-1")
	req := httptest.NewRequest(http.MethodGet, "/admin?email=private@example.test", nil).WithContext(ctx)
	handler.ServeHTTP(httptest.NewRecorder(), req)
	output := logs.String()
	for _, want := range []string{`"request_id":"req-1"`, `"actor_id":"actor-1"`, `"session_id":"session-1"`, `"job_id":"job-1"`} {
		if !strings.Contains(output, want) {
			t.Fatalf("log missing %s: %s", want, output)
		}
	}
	for _, forbidden := range []string{"private@example.test", "secret-token-hash", "email"} {
		if strings.Contains(output, forbidden) {
			t.Fatalf("log leaked %q: %s", forbidden, output)
		}
	}
}

func TestBoundedPathLeavesShortPathUnchanged(t *testing.T) {
	const path = "/admin/pages/1"
	if got := boundedPath(path); got != path {
		t.Fatalf("bounded path = %q, want %q", got, path)
	}
}

func TestBoundedPathDoesNotSplitUTF8(t *testing.T) {
	path := strings.Repeat("a", maxLoggedPathBytes-1) + "ğ" + "bbbb"
	bounded := boundedPath(path)
	if !utf8.ValidString(bounded) {
		t.Fatalf("bounded path is invalid UTF-8: %q", bounded)
	}
}
