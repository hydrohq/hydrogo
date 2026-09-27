package forms

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const turnstileVerifyURL = "https://challenges.cloudflare.com/turnstile/v0/siteverify"

type TurnstileVerifier struct {
	Secret string
	Client *http.Client
}

func (v TurnstileVerifier) Verify(ctx context.Context, token, remoteIP string) (bool, error) {
	if strings.TrimSpace(v.Secret) == "" || strings.TrimSpace(token) == "" || len(token) > 2048 {
		return false, errors.New("turnstile verification is not configured")
	}
	client := v.Client
	if client == nil {
		client = &http.Client{Timeout: 3 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	}
	values := url.Values{"secret": {v.Secret}, "response": {token}}
	if remoteIP = strings.TrimSpace(remoteIP); remoteIP != "" && len(remoteIP) <= 64 {
		values.Set("remoteip", remoteIP)
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, turnstileVerifyURL, strings.NewReader(values.Encode()))
	if err != nil {
		return false, err
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	response, err := client.Do(request)
	if err != nil {
		return false, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return false, errors.New("turnstile verification unavailable")
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, 4097))
	if err != nil || len(body) > 4096 {
		return false, errors.New("turnstile response is invalid")
	}
	var result struct {
		Success bool `json:"success"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return false, errors.New("turnstile response is invalid")
	}
	return result.Success, nil
}
