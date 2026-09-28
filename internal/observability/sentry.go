package observability

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
	"unicode/utf8"
)

var sentryFramePattern = regexp.MustCompile(`(?m)^([^\n]+)\n\s+([^\n]+\.go):(\d+) \+0x[0-9a-f]+$`)
var sentryKeyPattern = regexp.MustCompile(`^[A-Za-z0-9._-]{1,128}$`)
var sentryProjectPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{1,128}$`)

type sentryFrame struct {
	Function string `json:"function"`
	Filename string `json:"filename"`
	Lineno   int    `json:"lineno"`
}

type sentryEvent struct {
	EventID   string `json:"event_id"`
	Timestamp string `json:"timestamp"`
	Platform  string `json:"platform"`
	Level     string `json:"level"`
	Exception struct {
		Values []struct {
			Type       string `json:"type"`
			Value      string `json:"value"`
			Stacktrace struct {
				Frames []sentryFrame `json:"frames"`
			} `json:"stacktrace"`
		} `json:"values"`
	} `json:"exception"`
	Request struct {
		Method string `json:"method,omitempty"`
	} `json:"request,omitempty"`
}

// SentryReporter sends bounded, asynchronous panic events using the Sentry
// Store API. It intentionally excludes request URLs, headers, users, and panic
// values; source line contents are never included in stack frames.
type SentryReporter struct {
	endpoint  string
	publicKey string
	client    *http.Client
	queue     chan []byte
	ctx       context.Context
	cancel    context.CancelFunc
	stop      chan struct{}
	mu        sync.Mutex
	closed    bool
	done      chan struct{}
}

func NewSentryReporter(dsn string, client *http.Client, queueSize int) (*SentryReporter, error) {
	endpoint, publicKey, err := parseSentryDSN(dsn)
	if err != nil {
		return nil, err
	}
	if queueSize < 1 || queueSize > 4096 {
		queueSize = 64
	}
	if client == nil {
		client = &http.Client{Timeout: 3 * time.Second}
	} else {
		clientCopy := *client
		clientCopy.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
		client = &clientCopy
	}
	ctx, cancel := context.WithCancel(context.Background())
	reporter := &SentryReporter{endpoint: endpoint, publicKey: publicKey, client: client, queue: make(chan []byte, queueSize), ctx: ctx, cancel: cancel, stop: make(chan struct{}), done: make(chan struct{})}
	go reporter.run()
	return reporter, nil
}

func parseSentryDSN(value string) (string, string, error) {
	parsed, err := url.Parse(strings.TrimSpace(value))
	if err != nil || (parsed.Scheme != "https" && !(parsed.Scheme == "http" && (parsed.Hostname() == "localhost" || parsed.Hostname() == "127.0.0.1" || parsed.Hostname() == "::1"))) || parsed.Host == "" || parsed.User == nil || parsed.User.Username() == "" {
		return "", "", errors.New("invalid Sentry DSN")
	}
	parts := strings.Split(strings.Trim(parsed.Path, "/"), "/")
	if len(parts) == 0 || parts[len(parts)-1] == "" || parsed.RawQuery != "" || parsed.Fragment != "" {
		return "", "", errors.New("invalid Sentry DSN")
	}
	project := parts[len(parts)-1]
	publicKey := parsed.User.Username()
	if !sentryKeyPattern.MatchString(publicKey) || !sentryProjectPattern.MatchString(project) {
		return "", "", errors.New("invalid Sentry DSN")
	}
	prefix := strings.Join(parts[:len(parts)-1], "/")
	apiPath := "/api/" + url.PathEscape(project) + "/store/"
	if prefix != "" {
		apiPath = "/" + prefix + apiPath
	}
	parsed.Path = apiPath
	parsed.RawPath = ""
	parsed.RawQuery = ""
	parsed.Fragment = ""
	parsed.User = nil
	return parsed.String(), publicKey, nil
}

// CapturePanic drops events if the bounded queue is full, keeping recovery
// independent of provider availability and request latency.
func (s *SentryReporter) CapturePanic(_ context.Context, request *http.Request, panicType string, stack []byte) {
	if s == nil || request == nil {
		return
	}
	var id [16]byte
	if _, err := rand.Read(id[:]); err != nil {
		return
	}
	var event sentryEvent
	event.EventID = hex.EncodeToString(id[:])
	event.Timestamp = time.Now().UTC().Format(time.RFC3339Nano)
	event.Platform = "go"
	event.Level = "error"
	event.Request.Method = request.Method
	value := struct {
		Type       string `json:"type"`
		Value      string `json:"value"`
		Stacktrace struct {
			Frames []sentryFrame `json:"frames"`
		} `json:"stacktrace"`
	}{Type: bounded(panicType, 128), Value: "HTTP handler panic"}
	for _, match := range sentryFramePattern.FindAllSubmatch(stack, 64) {
		line := 0
		for _, digit := range match[3] {
			line = line*10 + int(digit-'0')
			if line > 1_000_000 {
				line = 0
				break
			}
		}
		value.Stacktrace.Frames = append(value.Stacktrace.Frames, sentryFrame{Function: bounded(string(match[1]), 256), Filename: bounded(filepath.Base(string(match[2])), 128), Lineno: line})
	}
	event.Exception.Values = append(event.Exception.Values, value)
	s.enqueue(event)
}

// CaptureHTTPError reports a bounded server-error event without the response
// body, URL, user identity, or request headers.
func (s *SentryReporter) CaptureHTTPError(request *http.Request, status int) {
	if s == nil || request == nil || status < http.StatusInternalServerError {
		return
	}
	var id [16]byte
	if _, err := rand.Read(id[:]); err != nil {
		return
	}
	var event sentryEvent
	event.EventID = hex.EncodeToString(id[:])
	event.Timestamp = time.Now().UTC().Format(time.RFC3339Nano)
	event.Platform = "go"
	event.Level = "error"
	event.Request.Method = request.Method
	value := struct {
		Type       string `json:"type"`
		Value      string `json:"value"`
		Stacktrace struct {
			Frames []sentryFrame `json:"frames"`
		} `json:"stacktrace"`
	}{Type: "HTTPError", Value: fmt.Sprintf("HTTP %d response", status)}
	event.Exception.Values = append(event.Exception.Values, value)
	s.enqueue(event)
}

func (s *SentryReporter) enqueue(event sentryEvent) {
	data, err := json.Marshal(event)
	if err != nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return
	}
	select {
	case s.queue <- data:
	default:
	}
}

func (s *SentryReporter) run() {
	defer close(s.done)
	for {
		var data []byte
		select {
		case <-s.stop:
			return
		case data = <-s.queue:
		}
		ctx, cancel := context.WithTimeout(s.ctx, 3*time.Second)
		request, err := http.NewRequestWithContext(ctx, http.MethodPost, s.endpoint, bytes.NewReader(data))
		if err == nil {
			request.Header.Set("Content-Type", "application/json")
			request.Header.Set("X-Sentry-Auth", "Sentry sentry_version=7, sentry_client=hydrogo/1, sentry_key="+s.publicKey)
			response, sendErr := s.client.Do(request)
			if response != nil {
				_ = response.Body.Close()
			}
			_ = sendErr
		}
		cancel()
	}
}

func (s *SentryReporter) Close() {
	if s == nil {
		return
	}
	s.mu.Lock()
	if !s.closed {
		s.closed = true
		s.cancel()
		close(s.stop)
	}
	s.mu.Unlock()
	<-s.done
}

func bounded(value string, max int) string {
	value = strings.ToValidUTF8(value, "")
	if len(value) > max {
		for len(value) > max {
			_, size := utf8.DecodeLastRuneInString(value)
			value = value[:len(value)-size]
		}
	}
	return value
}
