package config

import (
	"net/url"
	"strings"
	"time"
)

func validTokenTTL(raw string, choices []time.Duration) bool {
	value, err := time.ParseDuration(strings.TrimSpace(raw))
	if err != nil {
		return false
	}
	for _, choice := range choices {
		if value == choice {
			return true
		}
	}
	return false
}

func isHTTPOrigin(parsed *url.URL) bool {
	if parsed == nil || parsed.Host == "" || parsed.Hostname() == "" || parsed.User != nil {
		return false
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return false
	}
	return (parsed.Path == "" || parsed.Path == "/") && parsed.RawQuery == "" && parsed.Fragment == ""
}

func isHTTPBaseURL(parsed *url.URL) bool {
	if parsed == nil || parsed.Host == "" || parsed.Hostname() == "" || parsed.User != nil {
		return false
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return false
	}
	return len(parsed.Path) <= 512 && parsed.RawQuery == "" && parsed.Fragment == ""
}

func isHTTPSCallbackURL(raw string) bool {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	return err == nil && parsed != nil && parsed.Scheme == "https" && parsed.Host != "" && parsed.Hostname() != "" && parsed.User == nil && parsed.Opaque == "" && parsed.Fragment == ""
}

func validConfigComponent(value string) bool {
	value = strings.TrimSpace(value)
	if value == "" || len(value) > 128 {
		return false
	}
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '.' || r == '_' || r == '-' {
			continue
		}
		return false
	}
	return true
}
