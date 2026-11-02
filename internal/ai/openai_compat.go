package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"syscall"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/ai/prompts"
)

type OpenAICompatible struct {
	Endpoint string
	APIKey   string
	Model    string
	Client   *http.Client
}

func (p OpenAICompatible) Validate() error {
	if _, err := p.endpoint(); err != nil {
		return ErrProvider
	}
	if strings.TrimSpace(p.APIKey) == "" || strings.TrimSpace(p.Model) == "" {
		return ErrProvider
	}
	return nil
}

func (p OpenAICompatible) Suggest(ctx context.Context, request Request) (Suggestion, error) {
	endpoint, err := p.endpoint()
	if err != nil || p.Validate() != nil {
		return Suggestion{}, ErrProvider
	}
	client := p.Client
	if client == nil {
		allowLoopback := (endpoint.Scheme == "http")
		client = NewSSRFProtectedClient(20*time.Second, allowLoopback)
	} else {
		copy := *client
		if copy.Transport == nil {
			allowLoopback := (endpoint.Scheme == "http")
			copy.Transport = NewSSRFProtectedTransport(allowLoopback)
		}
		client = &copy
	}
	client.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	userContent, err := json.Marshal(struct {
		Locale       string `json:"locale"`
		TargetLocale string `json:"target_locale,omitempty"`
		Tone         string `json:"tone,omitempty"`
		Title        string `json:"title"`
		Body         string `json:"body"`
	}{request.Locale, request.TargetLocale, request.Tone, request.Title, request.Body})
	if err != nil {
		return Suggestion{}, ErrProvider
	}
	task := map[Operation]string{
		Refine:    "Improve clarity and grammar while preserving meaning.",
		SEO:       "Suggest a clear search-friendly title and body while avoiding keyword stuffing.",
		Summary:   "Condense the text while preserving its important facts.",
		Translate: "Translate the title and body into the requested target locale while preserving meaning.",
	}[request.Operation]
	prompt := prompts.MustRender("content_assist_user", map[string]string{
		"TASK":   task,
		"LOCALE": request.Locale,
		"DATA":   string(userContent),
	})
	sysContent := prompts.MustLoad("content_assist_system").Body
	payload, err := json.Marshal(struct {
		Model     string        `json:"model"`
		Messages  []chatMessage `json:"messages"`
		MaxTokens int           `json:"max_tokens"`
	}{p.Model, []chatMessage{{Role: "system", Content: sysContent}, {Role: "user", Content: prompt}}, 2048})
	if err != nil {
		return Suggestion{}, ErrProvider
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint.String(), bytes.NewReader(payload))
	if err != nil {
		return Suggestion{}, ErrProvider
	}
	req.Header.Set("Authorization", "Bearer "+p.APIKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	response, err := client.Do(req)
	if err != nil {
		return Suggestion{}, ErrProvider
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return Suggestion{}, ErrProvider
	}
	var envelope struct {
		Choices []struct {
			Message chatMessage `json:"message"`
		} `json:"choices"`
	}
	decoder := json.NewDecoder(io.LimitReader(response.Body, 32<<10))
	if err := decoder.Decode(&envelope); err != nil || len(envelope.Choices) != 1 {
		return Suggestion{}, ErrProvider
	}
	var suggestion Suggestion
	if err := json.Unmarshal([]byte(envelope.Choices[0].Message.Content), &suggestion); err != nil {
		return Suggestion{}, ErrProvider
	}
	return suggestion, nil
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

func (p OpenAICompatible) endpoint() (*url.URL, error) {
	parsed, err := endpointSecurityCheck(p.Endpoint)
	if err != nil {
		return nil, err
	}
	if !strings.HasSuffix(parsed.Path, "/chat/completions") {
		parsed.Path = strings.TrimRight(parsed.Path, "/") + "/chat/completions"
	}
	return parsed, nil
}

// isBlockedSSRFIP checks whether an IP address belongs to Private, Loopback,
// Link-Local (169.254.0.0/16, RFC 1918, RFC 3927) or other restricted ranges.
func isBlockedSSRFIP(ip net.IP, allowLoopback bool) bool {
	if ip == nil {
		return true
	}
	if ip.IsLoopback() {
		return !allowLoopback
	}
	if ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsUnspecified() {
		return true
	}
	if ip4 := ip.To4(); ip4 != nil {
		// 169.254.0.0/16 - Link-Local / AWS/GCP/Azure Metadata
		if ip4[0] == 169 && ip4[1] == 254 {
			return true
		}
		// 127.0.0.0/8 - Loopback
		if ip4[0] == 127 {
			return !allowLoopback
		}
		// 10.0.0.0/8 - Private RFC 1918
		if ip4[0] == 10 {
			return true
		}
		// 172.16.0.0/12 - Private RFC 1918
		if ip4[0] == 172 && ip4[1] >= 16 && ip4[1] <= 31 {
			return true
		}
		// 192.168.0.0/16 - Private RFC 1918
		if ip4[0] == 192 && ip4[1] == 168 {
			return true
		}
		// 100.64.0.0/10 - Carrier-Grade NAT
		if ip4[0] == 100 && ip4[1] >= 64 && ip4[1] <= 127 {
			return true
		}
		// 0.0.0.0/8
		if ip4[0] == 0 {
			return true
		}
	}
	return false
}

// NewSSRFProtectedTransport returns an http.Transport configured with a socket-level
// net.Dialer.Control hook that verifies resolved destination IP addresses before
// connecting. It blocks any connection to Private, Loopback, or Link-Local ranges.
func NewSSRFProtectedTransport(allowLoopback bool) *http.Transport {
	dialer := &net.Dialer{
		Timeout:   10 * time.Second,
		KeepAlive: 30 * time.Second,
		Control: func(network, address string, c syscall.RawConn) error {
			host, _, err := net.SplitHostPort(address)
			if err != nil {
				host = address
			}
			ip := net.ParseIP(host)
			if ip == nil {
				return errors.New("ssrf: invalid target IP address")
			}
			if isBlockedSSRFIP(ip, allowLoopback) {
				return fmt.Errorf("ssrf: connection to restricted address %s blocked", ip.String())
			}
			return nil
		},
	}

	return &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		DialContext:           dialer.DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          50,
		IdleConnTimeout:       60 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}
}

// NewSSRFProtectedClient returns an http.Client equipped with SSRF protection
// at both the transport and redirect layers.
func NewSSRFProtectedClient(timeout time.Duration, allowLoopback bool) *http.Client {
	if timeout <= 0 {
		timeout = 25 * time.Second
	}
	return &http.Client{
		Timeout:   timeout,
		Transport: NewSSRFProtectedTransport(allowLoopback),
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
}

// endpointSecurityCheck validates a provider endpoint before any request is
// made: https anywhere (excluding private/loopback/link-local literal IPs),
// http only on loopback (local Ollama and friends), and never with embedded credentials.
// This is the static SSRF gate for operator-provided HYDRO_AI_BASE_URL values on every adapter.
func endpointSecurityCheck(raw string) (*url.URL, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil, errors.New("empty AI provider endpoint")
	}
	parsed, err := url.Parse(trimmed)
	if err != nil || parsed.Hostname() == "" || parsed.User != nil {
		return nil, errors.New("invalid AI provider endpoint")
	}

	host := parsed.Hostname()
	ip := net.ParseIP(host)

	switch parsed.Scheme {
	case "https":
		// HTTPS cannot target literal private, loopback, or link-local IP addresses
		if ip != nil {
			if isBlockedSSRFIP(ip, false) {
				return nil, errors.New("https AI endpoint cannot target private, loopback, or link-local IP addresses")
			}
		}
		if host == "localhost" {
			return nil, errors.New("https AI endpoint cannot target localhost")
		}
	case "http":
		// HTTP is strictly restricted to local loopback (Ollama and local test runners)
		if ip == nil {
			if host != "localhost" {
				return nil, errors.New("unencrypted AI endpoint must be loopback")
			}
		} else if !ip.IsLoopback() {
			return nil, errors.New("unencrypted AI endpoint must be loopback")
		}
	default:
		return nil, errors.New("invalid AI provider endpoint scheme; must be https or loopback http")
	}
	return parsed, nil
}
