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
	"strings"
	"time"
)

var ErrInvalidEndpoint = errors.New("invalid webhook endpoint")

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
	attempts := d.MaxAttempts
	if attempts < 1 || attempts > 10 {
		attempts = 3
	}
	backoff := d.Backoff
	if backoff <= 0 {
		backoff = 250 * time.Millisecond
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
			request.Header.Set("X-Hydro-Signature", signature(body, endpoint.Secret))
			response, err := client.Do(request)
			if err == nil {
				_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, 1<<20))
				response.Body.Close()
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
	u, err := url.Parse(endpoint.URL)
	if err != nil || u.Host == "" || (u.Scheme != "https" && u.Scheme != "http") {
		return ErrInvalidEndpoint
	}
	return nil
}

func signature(body []byte, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write(body)
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}
