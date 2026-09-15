package sdkdist

import (
	"fmt"
	"net/url"
	"strings"
)

// ValidateDownloadURL accepts only broker URLs that can safely be used for an
// artifact download. Plain HTTP is allowed only for explicitly enabled local
// development and only on loopback hosts.
func ValidateDownloadURL(raw string, allowInsecureLocalhost bool) (*url.URL, error) {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return nil, fmt.Errorf("invalid artifact download URL")
	}
	if parsed.User != nil || parsed.Fragment != "" {
		return nil, fmt.Errorf("artifact download URL cannot contain credentials or fragments")
	}
	switch strings.ToLower(parsed.Scheme) {
	case "https":
	case "http":
		if !allowInsecureLocalhost || !isLoopbackHost(parsed.Hostname()) {
			return nil, fmt.Errorf("artifact download URL must use HTTPS")
		}
	default:
		return nil, fmt.Errorf("unsupported artifact download URL scheme %q", parsed.Scheme)
	}
	return parsed, nil
}

func isLoopbackHost(host string) bool {
	host = strings.ToLower(strings.TrimSpace(host))
	return host == "localhost" || host == "127.0.0.1" || host == "::1" || host == "[::1]"
}
