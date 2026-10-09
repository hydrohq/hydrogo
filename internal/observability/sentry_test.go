package observability

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/testhttp"
)

func TestSentryReporterSendsBoundedSanitizedPanicEvent(t *testing.T) {
	received := make(chan struct {
		body   []byte
		auth   string
		method string
	}, 1)
	const endpoint = "http://127.0.0.1:19094"
	client := testhttp.Client(testhttp.Handlers{"127.0.0.1:19094": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		received <- struct {
			body   []byte
			auth   string
			method string
		}{body, r.Header.Get("X-Sentry-Auth"), r.URL.Path}
		w.WriteHeader(http.StatusAccepted)
	})})
	reporter, err := NewSentryReporter(strings.Replace(endpoint, "http://", "http://public-key@", 1)+"/42", client, 2)
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodPost, "/private/person@example.com?token=secret", nil)
	request.Header.Set("Authorization", "Bearer very-secret")
	request.AddCookie(&http.Cookie{Name: "session", Value: "cookie-secret"})
	reporter.CapturePanic(context.Background(), request, "*errors.errorString", []byte("goroutine 1 [running]:\nexample.handler()\n /srv/app/handler.go:42 +0x10\n"))
	select {
	case got := <-received:
		if got.method != "/api/42/store/" || !strings.Contains(got.auth, "sentry_key=public-key") {
			t.Fatalf("endpoint/auth = %q %q", got.method, got.auth)
		}
		var event sentryEvent
		if err := json.Unmarshal(got.body, &event); err != nil {
			t.Fatalf("invalid event payload: %v", err)
		}
		if event.Platform != "go" || event.Level != "error" || event.Request.Method != http.MethodPost || len(event.Exception.Values) != 1 {
			t.Fatalf("event metadata = %+v", event)
		}
		encoded := string(got.body)
		for _, secret := range []string{"person@example.com", "token=secret", "very-secret", "cookie-secret", "HTTP handler panic"} {
			if secret != "HTTP handler panic" && strings.Contains(encoded, secret) {
				t.Errorf("event leaked request data %q: %s", secret, encoded)
			}
		}
		if !strings.Contains(encoded, "handler.go") || !strings.Contains(encoded, "*errors.errorString") {
			t.Errorf("event omitted panic diagnostics: %s", encoded)
		}
	case <-time.After(2 * time.Second):
		reporter.Close()
		t.Fatal("Sentry event was not delivered")
	}
	reporter.Close()
}

func TestSentryReporterRejectsUnsafeDSN(t *testing.T) {
	for _, dsn := range []string{"", "http://key@example.test/1", "https://example.test/1", "https://key@example.test/1?secret=x", "https://key@example.test/"} {
		if _, err := NewSentryReporter(dsn, nil, 1); err == nil {
			t.Errorf("accepted unsafe DSN %q", dsn)
		}
	}
}

func TestSentryReporterCapturesServerErrorWithoutRequestDetails(t *testing.T) {
	received := make(chan []byte, 1)
	const endpoint = "http://127.0.0.1:19095"
	client := testhttp.Client(testhttp.Handlers{"127.0.0.1:19095": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		received <- body
		w.WriteHeader(http.StatusAccepted)
	})})
	reporter, err := NewSentryReporter(strings.Replace(endpoint, "http://", "http://public-key@", 1)+"/project", client, 1)
	if err != nil {
		t.Fatal(err)
	}
	reporter.CaptureHTTPError(httptest.NewRequest(http.MethodGet, "/secret/path?password=secret", nil), http.StatusInternalServerError)
	var payload []byte
	select {
	case payload = <-received:
	case <-time.After(2 * time.Second):
		reporter.Close()
		t.Fatal("HTTP error was not delivered")
	}
	reporter.Close()
	var event sentryEvent
	if err := json.Unmarshal(payload, &event); err != nil {
		t.Fatal(err)
	}
	if event.Exception.Values[0].Type != "HTTPError" || event.Exception.Values[0].Value != "HTTP 500 response" || strings.Contains(event.Exception.Values[0].Value, "secret") {
		t.Fatalf("captured error = %+v", event.Exception.Values)
	}
}
