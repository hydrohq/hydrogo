package tenant

import (
	"net/http"
	"strings"
)

// HostResolver establishes tenant scope from an operator-owned host map. It
// never accepts a tenant ID from a request header or query parameter.
type HostResolver struct {
	hosts   map[string]Scope
	require bool
}

func NewHostResolver(hosts map[string]Scope, require bool) HostResolver {
	copyHosts := make(map[string]Scope, len(hosts))
	for host, scope := range hosts {
		copyHosts[normalizeHost(host)] = scope
	}
	return HostResolver{hosts: copyHosts, require: require}
}

func (r HostResolver) Enabled() bool { return len(r.hosts) > 0 || r.require }

// Middleware resolves the host before downstream auth/repository code. Health
// and readiness endpoints remain tenant-neutral for load balancers.
func (r HostResolver) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.URL.Path == "/healthz" || req.URL.Path == "/readyz" {
			next.ServeHTTP(w, req)
			return
		}
		scope, ok := r.hosts[normalizeHost(req.Host)]
		if !ok {
			if r.require {
				http.Error(w, "tenant unavailable", http.StatusMisdirectedRequest)
				return
			}
			next.ServeHTTP(w, req)
			return
		}
		next.ServeHTTP(w, req.WithContext(WithScope(req.Context(), scope)))
	})
}
func normalizeHost(raw string) string {
	host := strings.ToLower(strings.TrimSpace(raw))
	if host == "" {
		return ""
	}
	// Strip only a trailing ":port". Never trust an embedded colon followed by
	// arbitrary bytes: "host:443.evil.test" must not normalize back to "host",
	// otherwise a request with a grafted Host suffix resolves to the allowlisted
	// tenant. Anything malformed fails closed (never matches the allowlist).
	if i := strings.LastIndex(host, "]"); i != -1 {
		switch {
		case i == len(host)-1:
			return host[:i+1]
		case host[i+1] == ':' && isDigits(host[i+2:]):
			return host[:i+1]
		default:
			return ""
		}
	}
	if !strings.Contains(host, ":") {
		return host
	}
	before, after, _ := strings.Cut(host, ":")
	if isDigits(after) {
		return before
	}
	return ""
}

func isDigits(value string) bool {
	if value == "" {
		return false
	}
	for _, r := range value {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}
