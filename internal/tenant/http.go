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
	if parsed, _, ok := strings.Cut(host, ":"); ok {
		return parsed
	}
	return host
}
