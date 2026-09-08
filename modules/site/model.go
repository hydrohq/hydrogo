package site

import (
	"errors"
	"net/mail"
	"net/url"
	"strings"
	"time"
)

var (
	ErrInvalidSetting    = errors.New("invalid site setting")
	ErrInvalidNavigation = errors.New("invalid navigation item")
	ErrInvalidRedirect   = errors.New("invalid redirect")
	ErrInvalidSubmission = errors.New("invalid form submission")
)

type Setting struct {
	Key       string
	Value     string
	Public    bool
	UpdatedAt time.Time
}

type NavigationItem struct {
	ID        string
	Locale    string
	Label     string
	Href      string
	Position  int
	Visible   bool
	UpdatedAt time.Time
}

type Redirect struct {
	SourcePath string
	TargetPath string
	Status     int
	CreatedAt  time.Time
}

type Submission struct {
	ID           string
	FormKey      string
	Email        string
	Payload      string
	SourceIPHash string
	Status       string
	CreatedAt    time.Time
}

func ValidateSetting(value Setting) error {
	value.Key = strings.TrimSpace(value.Key)
	if value.Key == "" || len(value.Key) > 128 || strings.ContainsAny(value.Key, " \t\r\n") || len(value.Value) > 64<<10 {
		return ErrInvalidSetting
	}
	return nil
}

func ValidateNavigation(value NavigationItem) error {
	href := strings.TrimSpace(value.Href)
	if value.ID == "" || value.Locale == "" || strings.TrimSpace(value.Label) == "" || value.Position < 0 || href == "" {
		return ErrInvalidNavigation
	}
	parsed, err := url.Parse(href)
	if err != nil || parsed.Host != "" || parsed.IsAbs() && parsed.Scheme != "https" || strings.HasPrefix(strings.ToLower(href), "javascript:") {
		return ErrInvalidNavigation
	}
	return nil
}

func ValidateRedirect(value Redirect) error {
	source := normalizePath(value.SourcePath)
	target := normalizePath(value.TargetPath)
	if source == "" || target == "" || source == target || (value.Status != 301 && value.Status != 308) {
		return ErrInvalidRedirect
	}
	return nil
}

func ValidateSubmission(value Submission) error {
	if value.ID == "" || strings.TrimSpace(value.FormKey) == "" || len(value.Payload) > 64<<10 || strings.TrimSpace(value.SourceIPHash) == "" {
		return ErrInvalidSubmission
	}
	if _, err := mail.ParseAddress(value.Email); err != nil || len(value.Email) > 320 {
		return ErrInvalidSubmission
	}
	return nil
}

func normalizePath(value string) string {
	value = "/" + strings.Trim(strings.TrimSpace(value), "/")
	if value == "/" {
		return value
	}
	return strings.ToLower(value)
}
