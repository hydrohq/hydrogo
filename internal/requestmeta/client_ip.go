// Package requestmeta derives request metadata at the HTTP trust boundary.
package requestmeta

import (
	"fmt"
	"net"
	"net/http"
	"strings"
)

// ClientIP returns the directly connected peer address without its port.
// Forwarded headers are deliberately ignored until a separately configured
// trusted-proxy boundary exists; accepting them unconditionally lets callers
// choose their own rate-limit key and audit address.
func ClientIP(r *http.Request) string {
	if r == nil {
		return ""
	}
	address := strings.TrimSpace(r.RemoteAddr)
	if host, _, err := net.SplitHostPort(address); err == nil {
		return host
	}
	return address
}

// IPExtractor extracts the client IP address from a request, taking into
// account whether the server is behind a trusted proxy.
type IPExtractor struct {
	BehindProxy  bool
	TrustedCIDRs []*net.IPNet
}

// NewIPExtractor creates a new IPExtractor with the given trusted proxies.
func NewIPExtractor(behindProxy bool, trustedCIDRs []string) (*IPExtractor, error) {
	ext := &IPExtractor{BehindProxy: behindProxy}
	for _, cidr := range trustedCIDRs {
		cidr = strings.TrimSpace(cidr)
		if cidr == "" {
			continue
		}
		if !strings.Contains(cidr, "/") {
			if strings.Contains(cidr, ":") {
				cidr += "/128"
			} else {
				cidr += "/32"
			}
		}
		_, ipNet, err := net.ParseCIDR(cidr)
		if err != nil {
			return nil, fmt.Errorf("invalid trusted proxy CIDR %q: %w", cidr, err)
		}
		ext.TrustedCIDRs = append(ext.TrustedCIDRs, ipNet)
	}
	return ext, nil
}

// isTrusted checks if the given IP is a trusted proxy.
func (e *IPExtractor) isTrusted(ip net.IP) bool {
	if e.BehindProxy {
		return true
	}
	for _, cidr := range e.TrustedCIDRs {
		if cidr.Contains(ip) {
			return true
		}
	}
	return false
}

// Extract returns the best client IP address for the given request.
func (e *IPExtractor) Extract(r *http.Request) string {
	if r == nil {
		return ""
	}

	directIPStr := ClientIP(r)
	if directIPStr == "" {
		return ""
	}

	directIP := net.ParseIP(directIPStr)
	if directIP == nil {
		return directIPStr
	}

	if !e.isTrusted(directIP) {
		return directIPStr
	}

	if ip := r.Header.Get("True-Client-IP"); ip != "" {
		if parsed := net.ParseIP(strings.TrimSpace(ip)); parsed != nil {
			return parsed.String()
		}
	}
	if ip := r.Header.Get("CF-Connecting-IP"); ip != "" {
		if parsed := net.ParseIP(strings.TrimSpace(ip)); parsed != nil {
			return parsed.String()
		}
	}
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		ips := strings.Split(xff, ",")
		for i := len(ips) - 1; i >= 0; i-- {
			ipStr := strings.TrimSpace(ips[i])
			ip := net.ParseIP(ipStr)
			if ip != nil {
				if !e.isTrusted(ip) || i == 0 {
					return ip.String()
				}
			}
		}
	}

	return directIPStr
}
