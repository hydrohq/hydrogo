// Package requestmeta derives request metadata at the HTTP trust boundary.
package requestmeta

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strings"
)

type clientIPKey struct{}

// ClientIP returns the caller's address. When Middleware resolved it at the
// edge (behind a configured proxy) that value is used; otherwise it is the
// directly connected peer, without its port. Forwarded headers are never
// read here: only Middleware, which knows which peers are trusted proxies,
// may accept them.
func ClientIP(r *http.Request) string {
	if r == nil {
		return ""
	}
	if ip, ok := r.Context().Value(clientIPKey{}).(string); ok && ip != "" {
		return ip
	}
	return peerIP(r)
}

func peerIP(r *http.Request) string {
	address := strings.TrimSpace(r.RemoteAddr)
	if host, _, err := net.SplitHostPort(address); err == nil {
		return host
	}
	return address
}

// IPExtractor resolves the client address from forwarding headers, but only
// when the request arrived from a trusted proxy.
type IPExtractor struct {
	TrustedCIDRs []*net.IPNet
}

// loopbackProxies is the default trust set for behind_proxy without an
// explicit list: a reverse proxy on the same host.
var loopbackProxies = []string{"127.0.0.0/8", "::1/128"}

// NewIPExtractor returns an extractor that trusts trustedCIDRs. With no list
// and behindProxy false nothing is trusted and Extract returns the peer.
// With behindProxy true and no list, only a proxy on the same host
// (loopback) is trusted -- never "every peer", which would let any direct
// client choose its own address with X-Forwarded-For.
func NewIPExtractor(behindProxy bool, trustedCIDRs []string) (*IPExtractor, error) {
	ext := &IPExtractor{}
	if len(trustedCIDRs) == 0 {
		if !behindProxy {
			return ext, nil
		}
		trustedCIDRs = loopbackProxies
	}
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

func (e *IPExtractor) isTrusted(ip net.IP) bool {
	for _, cidr := range e.TrustedCIDRs {
		if cidr.Contains(ip) {
			return true
		}
	}
	return false
}

// Extract returns the client address. X-Forwarded-For is walked from the
// right, skipping trusted proxies; the first untrusted hop is the client.
// A request from an untrusted peer is taken at face value: its headers are
// ignored.
func (e *IPExtractor) Extract(r *http.Request) string {
	if r == nil {
		return ""
	}
	direct := peerIP(r)
	directIP := net.ParseIP(direct)
	if e == nil || directIP == nil || !e.isTrusted(directIP) {
		return direct
	}
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		hops := strings.Split(xff, ",")
		for i := len(hops) - 1; i >= 0; i-- {
			ip := net.ParseIP(strings.TrimSpace(hops[i]))
			if ip == nil {
				// A malformed hop ends the chain we can trust.
				return direct
			}
			if !e.isTrusted(ip) || i == 0 {
				// The first untrusted hop, or the originating host when
				// every hop is one of our own proxies.
				return ip.String()
			}
		}
	}
	if ip := net.ParseIP(strings.TrimSpace(r.Header.Get("X-Real-IP"))); ip != nil {
		return ip.String()
	}
	return direct
}

// Middleware resolves the client address once, at the edge, so every later
// handler, limiter and audit record that calls ClientIP agrees on it.
func Middleware(extractor *IPExtractor) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		if extractor == nil || len(extractor.TrustedCIDRs) == 0 {
			return next
		}
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := extractor.Extract(r)
			next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), clientIPKey{}, ip)))
		})
	}
}
