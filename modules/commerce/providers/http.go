// Package providers contains small, shared transport guards for payment
// provider adapters. It intentionally has no domain dependency.
package providers

import (
	"errors"
	"io"
	"net/http"
	"time"
)

const MaxResponseBytes = 1 << 20

var ErrResponseTooLarge = errors.New("provider response exceeds allowed size")

// HTTPClient rejects redirect-following by default. A payment adapter should
// only call the configured provider endpoint, never an attacker-controlled
// redirect destination.
func HTTPClient(client *http.Client) *http.Client {
	if client == nil {
		return &http.Client{Timeout: 10 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	}
	clone := *client
	if clone.Timeout <= 0 {
		clone.Timeout = 10 * time.Second
	}
	if clone.CheckRedirect == nil {
		clone.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	}
	return &clone
}

func ReadBounded(body io.Reader) ([]byte, error) {
	data, err := io.ReadAll(io.LimitReader(body, MaxResponseBytes+1))
	if err != nil {
		return nil, err
	}
	if len(data) > MaxResponseBytes {
		return nil, ErrResponseTooLarge
	}
	return data, nil
}
