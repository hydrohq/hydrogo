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
	host   string
	port   string
	origin string // scheme://host[:port] exactly as configured, used as the redirect target
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
			if !requestMatchesHost(r.Host, expected) && !isLoopbackDirectRequest(r) {
				// The www./apex twin of the public origin is the same site:
				// send the visitor to the canonical host instead of an error page.
				if isWWWSibling(r.Host, expected) {
					http.Redirect(w, r, expected.origin+r.URL.RequestURI(), http.StatusPermanentRedirect)
					return
				}
				http.Error(w, hostMismatchMessage(r.Host, expected), http.StatusMisdirectedRequest)
				return
			}
			next.ServeHTTP(w, r)
		})
	}, nil
}

// isLoopbackDirectRequest permits co-located server-side callers (such as an
// external SvelteKit frontend on the same VM) calling the HydroGo HTTP API
// over 127.0.0.1 / ::1 without spoofing the Host header (which Fetch forbids),
// while still blocking external peers or proxied mismatched Hosts (e.g. www).
func isLoopbackDirectRequest(r *http.Request) bool {
	peer := strings.TrimSpace(r.RemoteAddr)
	if peer == "" {
		return false
	}
	if h, _, err := net.SplitHostPort(peer); err == nil {
		peer = h
	}
	peer = strings.Trim(peer, "[]")
	peerIP := net.ParseIP(peer)
	if peerIP == nil || !peerIP.IsLoopback() {
		return false
	}
	host := strings.TrimSpace(r.Host)
	if host == "" {
		return false
	}
	if h, _, err := net.SplitHostPort(host); err == nil {
		host = h
	}
	host = strings.Trim(host, "[]")
	return host == "127.0.0.1" || host == "::1"
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
	return expectedHost{host: hostname, port: port, origin: parsed.Scheme + "://" + parsed.Host}, nil
}

// isWWWSibling reports whether value is the "www." twin of the expected host
// (public origin apex -> www.apex) or its apex (public origin www.x -> x), on
// the expected port. IP addresses and other hosts never qualify.
func isWWWSibling(value string, expected expectedHost) bool {
	if net.ParseIP(expected.host) != nil {
		return false
	}
	sibling := "www." + expected.host
	if strings.HasPrefix(expected.host, "www.") {
		sibling = strings.TrimPrefix(expected.host, "www.")
	}
	if sibling == "" || !strings.Contains(sibling, ".") {
		return false
	}
	return requestMatchesHost(value, expectedHost{host: sibling, port: expected.port})
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
