// Package webhooks delivers signed application events to configured external
// systems. It contains no persistence; durable outbox leasing remains the
// responsibility of the caller.
package webhooks

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

var ErrInvalidEndpoint = errors.New("invalid webhook endpoint")

const maxWebhookBackoff = 30 * time.Second

type Endpoint struct {
	ID     string
	URL    string
	Secret string
	Topics map[string]struct{}
}

type Event struct {
	ID             string `json:"id"`
	Topic          string `json:"topic"`
	AggregateID    string `json:"aggregate_id"`
	PayloadVersion int    `json:"payload_version"`
	Data           any    `json:"data,omitempty"`
}

type Dispatcher struct {
	Client      *http.Client
	Endpoints   []Endpoint
	MaxAttempts int
	Backoff     time.Duration
	Sleep       func(context.Context, time.Duration) error
}

type TestResult struct {
	Body      string        `json:"body"`
	Signature string        `json:"signature"`
	Timestamp string        `json:"timestamp"`
	Status    int           `json:"status"`
	Latency   time.Duration `json:"latency"`
	Response  string        `json:"response"`
}

// Test sends exactly one signed request and deliberately bypasses the durable event queue and retry loop.
func (d Dispatcher) Test(ctx context.Context, endpoint Endpoint, event Event) (TestResult, error) {
	if !endpoint.matches(event.Topic) {
		return TestResult{}, errors.New("endpoint does not subscribe to this topic")
	}
	if err := validateEndpoint(endpoint); err != nil {
		return TestResult{}, err
	}
	if event.ID == "" || event.PayloadVersion < 1 {
		return TestResult{}, errors.New("invalid test event")
	}
	body, err := json.Marshal(event)
	if err != nil {
		return TestResult{}, err
	}
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)
	sig := signatureAt(body, endpoint.Secret, timestamp)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint.URL, bytes.NewReader(body))
	if err != nil {
		return TestResult{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "HydroGo-Webhook/1")
	req.Header.Set("Idempotency-Key", event.ID)
	req.Header.Set("X-Hydro-Event", event.Topic)
	req.Header.Set("X-Hydro-Signature", sig)
	req.Header.Set("X-Hydro-Timestamp", timestamp)
	client := d.Client
	if client == nil {
		client = &http.Client{Timeout: 5 * time.Second}
	}
	start := time.Now()
	response, err := safeClient(client).Do(req)
	latency := time.Since(start)
	if err != nil {
		return TestResult{Body: string(body), Signature: sig, Timestamp: timestamp, Latency: latency}, err
	}
	defer response.Body.Close()
	preview, readErr := io.ReadAll(io.LimitReader(response.Body, 2049))
	if readErr != nil {
		return TestResult{Body: string(body), Signature: sig, Timestamp: timestamp, Status: response.StatusCode, Latency: latency}, readErr
	}
	if len(preview) > 2048 {
		preview = preview[:2048]
	}
	result := TestResult{Body: string(body), Signature: sig, Timestamp: timestamp, Status: response.StatusCode, Latency: latency, Response: string(preview)}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return result, fmt.Errorf("webhook returned status %d", response.StatusCode)
	}
	return result, nil
}

func (d Dispatcher) Dispatch(ctx context.Context, event Event) error {
	if strings.TrimSpace(event.ID) == "" || strings.TrimSpace(event.Topic) == "" || event.PayloadVersion < 1 {
		return errors.New("invalid webhook event")
	}
	body, err := json.Marshal(event)
	if err != nil {
		return err
	}
	client := d.Client
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}
	client = safeClient(client)
	attempts := d.MaxAttempts
	if attempts < 1 || attempts > 10 {
		attempts = 3
	}
	backoff := d.Backoff
	if backoff <= 0 {
		backoff = 250 * time.Millisecond
	}
	if backoff > maxWebhookBackoff {
		backoff = maxWebhookBackoff
	}
	sleep := d.Sleep
	if sleep == nil {
		sleep = func(ctx context.Context, delay time.Duration) error {
			timer := time.NewTimer(delay)
			defer timer.Stop()
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-timer.C:
				return nil
			}
		}
	}
	var failures []string
	for _, endpoint := range d.Endpoints {
		if !endpoint.matches(event.Topic) {
			continue
		}
		if err := validateEndpoint(endpoint); err != nil {
			return err
		}
		var delivered bool
		var lastErr error
		for attempt := 1; attempt <= attempts; attempt++ {
			request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint.URL, bytes.NewReader(body))
			if err != nil {
				return err
			}
			request.Header.Set("Content-Type", "application/json")
			request.Header.Set("User-Agent", "HydroGo-Webhook/1")
			request.Header.Set("Idempotency-Key", event.ID)
			request.Header.Set("X-Hydro-Event", event.Topic)
			timestamp := strconv.FormatInt(time.Now().Unix(), 10)
			request.Header.Set("X-Hydro-Signature", signatureAt(body, endpoint.Secret, timestamp))
			request.Header.Set("X-Hydro-Timestamp", timestamp)
			response, err := client.Do(request)
			if err == nil {
				_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, 1<<20))
				_ = response.Body.Close()
				if response.StatusCode >= 200 && response.StatusCode < 300 {
					delivered = true
					break
				}
				err = fmt.Errorf("unexpected status %s", response.Status)
			}
			lastErr = err
			if attempt < attempts {
				if sleepErr := sleep(ctx, backoff*time.Duration(1<<(attempt-1))); sleepErr != nil {
					return sleepErr
				}
			}
		}
		if !delivered {
			failures = append(failures, endpoint.ID+": "+lastErr.Error())
		}
	}
	if len(failures) > 0 {
		return fmt.Errorf("webhook delivery failed: %s", strings.Join(failures, "; "))
	}
	return nil
}

func (e Endpoint) matches(topic string) bool {
	_, ok := e.Topics[topic]
	return ok
}

func validateEndpoint(endpoint Endpoint) error {
	if strings.TrimSpace(endpoint.ID) == "" || strings.TrimSpace(endpoint.Secret) == "" {
		return ErrInvalidEndpoint
	}
	if err := validateURL(endpoint.URL); err != nil {
		return ErrInvalidEndpoint
	}
	return nil
}

func validateURL(raw string) error {
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" || u.Hostname() == "" || u.User != nil || u.Opaque != "" || u.Fragment != "" {
		return ErrInvalidEndpoint
	}
	if u.Scheme != "https" && u.Scheme != "http" {
		return ErrInvalidEndpoint
	}
	if u.Port() == "" && strings.Contains(u.Host, ":") && !strings.HasPrefix(u.Host, "[") {
		return ErrInvalidEndpoint
	}
	return nil
}

// safeClient preserves the caller's transport and redirect policy while
// preventing a valid public webhook URL from redirecting to an invalid target.
// Redirects are a second URL boundary and must receive the same validation as
// the configured endpoint.
func safeClient(client *http.Client) *http.Client {
	copy := *client
	previous := copy.CheckRedirect
	copy.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		if previous != nil {
			if err := previous(req, via); err != nil {
				return err
			}
		}
		return validateURL(req.URL.String())
	}
	return &copy
}

func signature(body []byte, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write(body)
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}

func signatureAt(body []byte, secret, timestamp string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = io.WriteString(mac, timestamp+".")
	_, _ = mac.Write(body)
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}
