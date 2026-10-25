package security

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"golang.org/x/net/idna"
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
				http.Error(w, hostMismatchMessage(r.Host, expected), http.StatusMisdirectedRequest)
				return
			}
			next.ServeHTTP(w, r)
		})
	}, nil
}

// hostMismatchMessage says what was expected, so a reverse proxy or a Docker
// network alias that forgot to forward the public Host is a one-line fix and
// not a guessing game. The expected value is the public origin, which is not
// secret; the received one is shortened and stripped of anything unprintable.
func hostMismatchMessage(got string, expected expectedHost) string {
	clean := strings.Map(func(r rune) rune {
		if r < 0x20 || r == 0x7f {
			return -1
		}
		return r
	}, got)
	if len(clean) > 80 {
		clean = clean[:80] + "..."
	}
	return fmt.Sprintf("invalid host %q: this server only answers to %s:%s (its public_origin). Behind a proxy or inside Docker, send that host in the Host header.", clean, expected.host, expected.port)
}

func parseExpectedHost(origin string) (expectedHost, error) {
	parsed, err := url.Parse(origin)
	if err != nil || parsed.Host == "" || parsed.User != nil || parsed.Path != "" && parsed.Path != "/" || parsed.RawQuery != "" || parsed.Fragment != "" || parsed.Scheme != "http" && parsed.Scheme != "https" {
		return expectedHost{}, fmt.Errorf("invalid public origin %q", origin)
	}
	hostname, err := canonicalHostname(parsed.Hostname())
	if err != nil {
		return expectedHost{}, err
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
	var err error
	hostname, err = canonicalHostname(hostname)
	if err != nil {
		return false
	}
	if port == "" {
		return false
	}
	if _, err := strconv.Atoi(port); err != nil {
		return false
	}
	return hostname == expected.host && port == expected.port
}

func canonicalHostname(value string) (string, error) {
	value = strings.TrimSuffix(strings.TrimSpace(strings.ToLower(value)), ".")
	if value == "" {
		return "", errors.New("public origin hostname is required")
	}
	if ip := net.ParseIP(value); ip != nil {
		return ip.String(), nil
	}
	ascii, err := idna.Lookup.ToASCII(value)
	if err != nil || ascii == "" {
		return "", fmt.Errorf("invalid hostname %q", value)
	}
	return strings.TrimSuffix(strings.ToLower(ascii), "."), nil
}
