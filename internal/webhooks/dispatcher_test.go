package webhooks

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestDispatcherSignsAndRetriesIdempotently(t *testing.T) {
	var attempts atomic.Int32
	client := &http.Client{Transport: webhookRoundTripperFunc(func(r *http.Request) (*http.Response, error) {
		attempt := attempts.Add(1)
		body, _ := io.ReadAll(r.Body)
		timestamp := r.Header.Get("X-Hydro-Timestamp")
		if _, err := strconv.ParseInt(timestamp, 10, 64); err != nil {
			t.Error("missing or invalid timestamp")
		}
		mac := hmac.New(sha256.New, []byte("secret"))
		_, _ = io.WriteString(mac, timestamp+".")
		_, _ = mac.Write(body)
		if r.Header.Get("X-Hydro-Signature") != "sha256="+hex.EncodeToString(mac.Sum(nil)) {
			t.Error("invalid signature")
		}
		if r.Header.Get("Idempotency-Key") != "evt-1" {
			t.Error("missing idempotency key")
		}
		status, responseBody := http.StatusNoContent, ""
		if attempt < 2 {
			status, responseBody = http.StatusBadGateway, "retry"
		}
		return &http.Response{StatusCode: status, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(responseBody)), Request: r}, nil
	})}
	d := Dispatcher{Client: client, Endpoints: []Endpoint{{ID: "erp", URL: "https://erp.example.test/hook", Secret: "secret", Topics: map[string]struct{}{"order.paid": {}}}}, MaxAttempts: 3, Sleep: func(context.Context, time.Duration) error { return nil }}
	if err := d.Dispatch(context.Background(), Event{ID: "evt-1", Topic: "order.paid", AggregateID: "o-1", PayloadVersion: 1, Data: map[string]string{"status": "paid"}}); err != nil {
		t.Fatal(err)
	}
	if attempts.Load() != 2 {
		t.Fatalf("attempts = %d", attempts.Load())
	}
}

func TestDispatcherRejectsInvalidEndpoint(t *testing.T) {
	for _, badURL := range []string{
		"file:///secret", "http://user:pass@example.com/hook", "ftp://example.com", "not-a-url",
		"http://example.com:bad/hook", "http://example.com/hook#fragment",
		// Literal internal/non-routable addresses: an admin pasting one of
		// these (by mistake or otherwise) must not get a webhook that calls
		// back into this host's own network, including the cloud metadata
		// address a compromised endpoint config could use to steal
		// instance credentials.
		"http://127.0.0.1/hook", "http://localhost/hook", "http://[::1]/hook",
		"http://169.254.169.254/latest/meta-data/", "http://10.0.0.5/hook",
		"http://192.168.1.1/hook", "http://172.16.0.1/hook", "http://0.0.0.0/hook",
	} {
		err := (Dispatcher{Endpoints: []Endpoint{{ID: "bad", URL: badURL, Secret: "x", Topics: map[string]struct{}{"x": {}}}}}).Dispatch(context.Background(), Event{ID: "e", Topic: "x", PayloadVersion: 1})
		if err == nil || !strings.Contains(err.Error(), "invalid webhook endpoint") {
			t.Fatalf("badURL %q error = %v, want invalid webhook endpoint", badURL, err)
		}
	}
}

// TestAllowInternalAddressesPermitsALoopbackEndpoint proves the opt-out
// exists and works: a caller with a different trust model than the default
// external-webhooks case (e.g. a same-deployment synchronous extension
// hook) can still reach a loopback service.
func TestAllowInternalAddressesPermitsALoopbackEndpoint(t *testing.T) {
	client := &http.Client{Transport: webhookRoundTripperFunc(func(r *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader("")), Header: http.Header{}}, nil
	})}
	d := Dispatcher{Client: client, AllowInternalAddresses: true, Endpoints: []Endpoint{{ID: "sidecar", URL: "http://127.0.0.1:9999/hook", Secret: "secret", Topics: map[string]struct{}{"x": {}}}}, Sleep: func(context.Context, time.Duration) error { return nil }}
	if err := d.Dispatch(context.Background(), Event{ID: "e", Topic: "x", PayloadVersion: 1}); err != nil {
		t.Fatalf("Dispatch with AllowInternalAddresses = %v, want nil", err)
	}
}

// TestDispatcherAllowsAnUnresolvedHostname proves the internal-address
// rejection above only applies to literal IPs, not every hostname -- a
// normal external webhook URL that happens to share a name pattern with
// nothing internal must never be refused just because its name looks like
// a domain. (A real DNS lookup is deliberately not performed; see
// rejectInternalHost's doc comment for why.)
func TestDispatcherAllowsAnUnresolvedHostname(t *testing.T) {
	if err := validateURL("https://erp.example.test/hook", false); err != nil {
		t.Fatalf("validateURL(ordinary hostname) = %v, want nil", err)
	}
}

func TestDispatcherClampsBackoff(t *testing.T) {
	client := &http.Client{Transport: webhookRoundTripperFunc(func(r *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusBadGateway, Header: make(http.Header), Body: io.NopCloser(strings.NewReader("retry")), Request: r}, nil
	})}
	var delays []time.Duration
	d := Dispatcher{
		Client:      client,
		Endpoints:   []Endpoint{{ID: "erp", URL: "https://erp.example.test/hook", Secret: "secret", Topics: map[string]struct{}{"order.paid": {}}}},
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
	client := &http.Client{Transport: webhookRoundTripperFunc(func(r *http.Request) (*http.Response, error) {
		if r.URL.Host == "target.example.test" {
			targetCalled.Store(true)
		}
		header := make(http.Header)
		header.Set("Location", "http://user:pass@target.example.test/hook")
		return &http.Response{StatusCode: http.StatusFound, Header: header, Body: io.NopCloser(strings.NewReader("redirect")), Request: r}, nil
	})}
	err := (Dispatcher{Client: client, Endpoints: []Endpoint{{ID: "redirect", URL: "https://redirect.example.test/hook", Secret: "secret", Topics: map[string]struct{}{"x": {}}}}}).Dispatch(context.Background(), Event{ID: "e", Topic: "x", PayloadVersion: 1})
	if err == nil || !strings.Contains(err.Error(), "invalid webhook endpoint") {
		t.Fatalf("redirect error = %v, want invalid webhook endpoint", err)
	}
	if targetCalled.Load() {
		t.Fatal("invalid redirect target received a request")
	}
}

type webhookRoundTripperFunc func(*http.Request) (*http.Response, error)

func (f webhookRoundTripperFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}
