package security

import (
	"bytes"
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRecoveryMiddlewareReturnsInternalServerError(t *testing.T) {
	handler := RecoveryMiddleware(slog.Default())(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		panic("unexpected failure")
	}))
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil).WithContext(context.Background())
	res := httptest.NewRecorder()
	handler.ServeHTTP(res, req)
	if res.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", res.Code, http.StatusInternalServerError)
	}
}

func TestRecoveryMiddlewarePreservesAbortHandler(t *testing.T) {
	handler := RecoveryMiddleware(nil)(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		panic(http.ErrAbortHandler)
	}))
	defer func() {
		if recover() != http.ErrAbortHandler {
			t.Fatal("expected http.ErrAbortHandler to be re-panicked")
		}
	}()
	handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))
}

func TestRecoveryMiddlewareHandlesUncomparablePanicValues(t *testing.T) {
	handler := RecoveryMiddleware(nil)(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		panic([]string{"unexpected"})
	}))
	res := httptest.NewRecorder()
	handler.ServeHTTP(res, httptest.NewRequest(http.MethodGet, "/", nil))
	if res.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", res.Code, http.StatusInternalServerError)
	}
}

func TestRecoveryMiddlewareDoesNotLogPanicValue(t *testing.T) {
	var logs bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logs, nil))
	handler := RecoveryMiddleware(logger)(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		panic("secret-token-value")
	}))
	handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))
	if bytes.Contains(logs.Bytes(), []byte("secret-token-value")) {
		t.Fatalf("panic value leaked into logs: %q", logs.String())
	}
	if !bytes.Contains(logs.Bytes(), []byte(`"panic_type":"string"`)) {
		t.Fatalf("panic type was not recorded: %q", logs.String())
	}
}
