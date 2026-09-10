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
	err := (Dispatcher{Endpoints: []Endpoint{{ID: "bad", URL: "file:///secret", Secret: "x", Topics: map[string]struct{}{"x": {}}}}}).Dispatch(context.Background(), Event{ID: "e", Topic: "x", PayloadVersion: 1})
	if !strings.Contains(err.Error(), "invalid webhook endpoint") {
		t.Fatalf("error = %v", err)
	}
}
