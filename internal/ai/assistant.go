// Package ai contains the provider-neutral, opt-in content assistance boundary.
package ai

import (
	"context"
	"errors"
	"regexp"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

var (
	ErrDisabled       = errors.New("AI assistance is disabled")
	ErrConsent        = errors.New("explicit content-sharing consent is required")
	ErrInvalidRequest = errors.New("invalid AI assistance request")
	ErrProvider       = errors.New("AI provider unavailable")
)

const (
	MaxInputBytes  = 24 << 10
	MaxOutputBytes = 16 << 10
)

type Operation string

const (
	Refine  Operation = "refine"
	SEO     Operation = "seo"
	Summary Operation = "summary"
)

type Request struct {
	Operation Operation
	Locale    string
	Title     string
	Body      string
	Consent   bool
}

type Suggestion struct {
	Title string
	Body  string
}

type Provider interface {
	Suggest(context.Context, Request) (Suggestion, error)
}

type Assistant struct {
	Enabled  bool
	Provider Provider
	Timeout  time.Duration
}

func (a Assistant) Suggest(ctx context.Context, request Request) (Suggestion, error) {
	if !a.Enabled || a.Provider == nil {
		return Suggestion{}, ErrDisabled
	}
	if !request.Consent {
		return Suggestion{}, ErrConsent
	}
	request.Operation = Operation(strings.TrimSpace(string(request.Operation)))
	if request.Operation != Refine && request.Operation != SEO && request.Operation != Summary {
		return Suggestion{}, ErrInvalidRequest
	}
	request.Locale = boundedText(request.Locale, 32)
	request.Title = redact(boundedText(request.Title, 2000))
	request.Body = redact(boundedText(request.Body, MaxInputBytes))
	if len(request.Title)+len(request.Body) == 0 || len(request.Title)+len(request.Body) > MaxInputBytes {
		return Suggestion{}, ErrInvalidRequest
	}
	timeout := a.Timeout
	if timeout <= 0 || timeout > 30*time.Second {
		timeout = 15 * time.Second
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	suggestion, err := a.Provider.Suggest(ctx, request)
	if err != nil {
		return Suggestion{}, ErrProvider
	}
	suggestion.Title = boundedText(suggestion.Title, 2000)
	suggestion.Body = boundedText(suggestion.Body, MaxOutputBytes)
	if len(suggestion.Title)+len(suggestion.Body) == 0 || len(suggestion.Title)+len(suggestion.Body) > MaxOutputBytes {
		return Suggestion{}, ErrProvider
	}
	return suggestion, nil
}

var (
	emailPattern  = regexp.MustCompile(`(?i)\b[A-Z0-9._%+\-]+@[A-Z0-9.\-]+\.[A-Z]{2,}\b`)
	secretPattern = regexp.MustCompile(`(?i)\b(?:sk-[A-Z0-9_-]{16,}|gh[pousr]_[A-Z0-9]{20,}|AKIA[A-Z0-9]{16})\b`)
)

func redact(value string) string {
	value = emailPattern.ReplaceAllString(value, "[redacted-email]")
	return secretPattern.ReplaceAllString(value, "[redacted-secret]")
}

func boundedText(value string, max int) string {
	value = strings.TrimSpace(strings.ToValidUTF8(value, ""))
	var clean strings.Builder
	clean.Grow(min(len(value), max))
	for _, r := range value {
		if unicode.IsControl(r) && r != '\n' && r != '\t' {
			continue
		}
		if clean.Len()+utf8.RuneLen(r) > max {
			break
		}
		clean.WriteRune(r)
	}
	return clean.String()
}
