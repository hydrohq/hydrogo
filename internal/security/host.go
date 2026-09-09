package security

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

type expectedHost struct {
	host string
	port string
}

// NewHostMiddleware binds requests to the configured canonical origin. An
// empty origin keeps local development permissive; a configured origin rejects
// unexpected Host headers without trusting forwarded proxy headers.
func NewHostMiddleware(publicOrigin string) (func(http.Handler) http.Handler, error) {
	publicOrigin = strings.TrimSpace(publicOrigin)
	if publicOrigin == "" {
		return func(next http.Handler) http.Handler { return next }, nil
	}
	expected, err := parseExpectedHost(publicOrigin)
	if err != nil {
		return nil, err
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !requestMatchesHost(r.Host, expected) {
				http.Error(w, "invalid host", http.StatusMisdirectedRequest)
				return
			}
			next.ServeHTTP(w, r)
		})
	}, nil
}

func parseExpectedHost(origin string) (expectedHost, error) {
	parsed, err := url.Parse(origin)
	if err != nil || parsed.Host == "" || parsed.User != nil || parsed.Path != "" && parsed.Path != "/" || parsed.RawQuery != "" || parsed.Fragment != "" || parsed.Scheme != "http" && parsed.Scheme != "https" {
		return expectedHost{}, fmt.Errorf("invalid public origin %q", origin)
	}
	hostname := strings.TrimSuffix(strings.ToLower(parsed.Hostname()), ".")
	if hostname == "" {
		return expectedHost{}, errors.New("public origin hostname is required")
	}
	port := parsed.Port()
	if port == "" {
		if parsed.Scheme == "https" {
			port = "443"
		} else {
			port = "80"
		}
	}
	portNumber, err := strconv.Atoi(port)
	if err != nil || portNumber < 1 || portNumber > 65535 {
		return expectedHost{}, fmt.Errorf("invalid public origin port %q", port)
	}
	return expectedHost{host: hostname, port: port}, nil
}

func requestMatchesHost(value string, expected expectedHost) bool {
	value = strings.TrimSpace(value)
	if value == "" {
		return false
	}
	hostname := value
	port := ""
	if strings.Contains(value, ":") {
		parsedHost, parsedPort, err := net.SplitHostPort(value)
		if err != nil {
			return false
		}
		hostname, port = parsedHost, parsedPort
	}
	if port == "" {
		port = expected.port
	}
	hostname = strings.TrimSuffix(strings.ToLower(hostname), ".")
	if port == "" {
		return false
	}
	if _, err := strconv.Atoi(port); err != nil {
		return false
	}
	return hostname == expected.host && port == expected.port
}
