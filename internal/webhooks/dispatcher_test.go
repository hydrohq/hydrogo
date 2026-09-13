package webhooks

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestDispatcherSignsAndRetriesIdempotently(t *testing.T) {
	var attempts atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts.Add(1)
		body, _ := io.ReadAll(r.Body)
		mac := hmac.New(sha256.New, []byte("secret"))
		_, _ = mac.Write(body)
		if r.Header.Get("X-Hydro-Signature") != "sha256="+hex.EncodeToString(mac.Sum(nil)) {
			t.Error("invalid signature")
		}
		if r.Header.Get("Idempotency-Key") != "evt-1" {
			t.Error("missing idempotency key")
		}
		if attempts.Load() < 2 {
			http.Error(w, "retry", http.StatusBadGateway)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()
	d := Dispatcher{Endpoints: []Endpoint{{ID: "erp", URL: server.URL, Secret: "secret", Topics: map[string]struct{}{"order.paid": {}}}}, MaxAttempts: 3, Sleep: func(context.Context, time.Duration) error { return nil }}
	if err := d.Dispatch(context.Background(), Event{ID: "evt-1", Topic: "order.paid", AggregateID: "o-1", PayloadVersion: 1, Data: map[string]string{"status": "paid"}}); err != nil {
		t.Fatal(err)
	}
	if attempts.Load() != 2 {
		t.Fatalf("attempts = %d", attempts.Load())
	}
}

func TestDispatcherRejectsInvalidEndpoint(t *testing.T) {
	for _, badURL := range []string{"file:///secret", "http://user:pass@example.com/hook", "ftp://example.com", "not-a-url", "http://example.com:bad/hook", "http://example.com/hook#fragment"} {
		err := (Dispatcher{Endpoints: []Endpoint{{ID: "bad", URL: badURL, Secret: "x", Topics: map[string]struct{}{"x": {}}}}}).Dispatch(context.Background(), Event{ID: "e", Topic: "x", PayloadVersion: 1})
		if err == nil || !strings.Contains(err.Error(), "invalid webhook endpoint") {
			t.Fatalf("badURL %q error = %v, want invalid webhook endpoint", badURL, err)
		}
	}
}

func TestDispatcherClampsBackoff(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "retry", http.StatusBadGateway)
	}))
	defer server.Close()
	var delays []time.Duration
	d := Dispatcher{
		Client:      server.Client(),
		Endpoints:   []Endpoint{{ID: "erp", URL: server.URL, Secret: "secret", Topics: map[string]struct{}{"order.paid": {}}}},
		MaxAttempts: 2,
		Backoff:     time.Duration(1<<63 - 1),
		Sleep: func(_ context.Context, delay time.Duration) error {
			delays = append(delays, delay)
			return nil
		},
	}
	if err := d.Dispatch(context.Background(), Event{ID: "evt", Topic: "order.paid", AggregateID: "order", PayloadVersion: 1}); err == nil {
		t.Fatal("expected failed delivery")
	}
	if len(delays) != 1 || delays[0] != maxWebhookBackoff {
		t.Fatalf("backoff delays = %v, want [%v]", delays, maxWebhookBackoff)
	}
}

func TestDispatcherRejectsInvalidRedirectTarget(t *testing.T) {
	targetCalled := atomic.Bool{}
	target := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		targetCalled.Store(true)
	}))
	defer target.Close()
	redirect := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, strings.Replace(target.URL, "http://", "http://user:pass@", 1), http.StatusFound)
	}))
	defer redirect.Close()

	err := (Dispatcher{Endpoints: []Endpoint{{ID: "redirect", URL: redirect.URL, Secret: "secret", Topics: map[string]struct{}{"x": {}}}}}).Dispatch(context.Background(), Event{ID: "e", Topic: "x", PayloadVersion: 1})
	if err == nil || !strings.Contains(err.Error(), "invalid webhook endpoint") {
		t.Fatalf("redirect error = %v, want invalid webhook endpoint", err)
	}
	if targetCalled.Load() {
		t.Fatal("invalid redirect target received a request")
	}
}
