package tenant

import (
	"context"
	"net/http"
	"strings"
	"sync"
	"time"
)

type cachedScope struct {
	scope     Scope
	found     bool
	expiresAt time.Time
}

type hostRegistry struct {
	mu          sync.RWMutex
	hosts       map[string]Scope
	cache       map[string]cachedScope
	fallback    func(ctx context.Context, host string) (Scope, bool)
	fallbackTTL time.Duration
}

// HostResolver establishes tenant scope from an operator-owned host map or
// dynamic registry. It never accepts a tenant ID from a request header or
// query parameter.
type HostResolver struct {
	registry *hostRegistry
	require  bool
}

func NewHostResolver(hosts map[string]Scope, require bool) HostResolver {
	copyHosts := make(map[string]Scope, len(hosts))
	for host, scope := range hosts {
		if norm := normalizeHost(host); norm != "" {
			copyHosts[norm] = scope
		}
	}
	return HostResolver{
		registry: &hostRegistry{
			hosts:       copyHosts,
			cache:       make(map[string]cachedScope),
			fallbackTTL: 5 * time.Minute,
		},
		require: require,
	}
}

// SetFallback configures an optional dynamic lookup (such as a database query)
// with an in-memory TTL cache for hosts not present in the static map.
func (r *HostResolver) SetFallback(fn func(ctx context.Context, host string) (Scope, bool), ttl time.Duration) {
	if r.registry == nil {
		r.registry = &hostRegistry{
			hosts: make(map[string]Scope),
			cache: make(map[string]cachedScope),
		}
	}
	if ttl <= 0 {
		ttl = 5 * time.Minute
	}
	r.registry.mu.Lock()
	r.registry.fallback = fn
	r.registry.fallbackTTL = ttl
	r.registry.cache = make(map[string]cachedScope)
	r.registry.mu.Unlock()
}

// SetHost registers or updates a host-to-tenant mapping at runtime without
// requiring a server restart.
func (r HostResolver) SetHost(raw string, scope Scope) bool {
	norm := normalizeHost(raw)
	if norm == "" || r.registry == nil {
		return false
	}
	r.registry.mu.Lock()
	r.registry.hosts[norm] = scope
	delete(r.registry.cache, norm)
	r.registry.mu.Unlock()
	return true
}

// RemoveHost removes a host mapping at runtime and invalidates any cached entry.
func (r HostResolver) RemoveHost(raw string) {
	norm := normalizeHost(raw)
	if norm == "" || r.registry == nil {
		return
	}
	r.registry.mu.Lock()
	delete(r.registry.hosts, norm)
	delete(r.registry.cache, norm)
	r.registry.mu.Unlock()
}

// Hosts returns a snapshot copy of the currently registered static/runtime hosts.
func (r HostResolver) Hosts() map[string]Scope {
	if r.registry == nil {
		return nil
	}
	r.registry.mu.RLock()
	defer r.registry.mu.RUnlock()
	out := make(map[string]Scope, len(r.registry.hosts))
	for k, v := range r.registry.hosts {
		out[k] = v
	}
	return out
}

func (r HostResolver) Enabled() bool {
	if r.require {
		return true
	}
	if r.registry == nil {
		return false
	}
	r.registry.mu.RLock()
	defer r.registry.mu.RUnlock()
	return len(r.registry.hosts) > 0 || r.registry.fallback != nil
}

// Resolve looks up the tenant Scope for raw host using the runtime map and
// optional TTL-cached fallback.
func (r HostResolver) Resolve(ctx context.Context, raw string) (Scope, bool) {
	norm := normalizeHost(raw)
	if norm == "" || r.registry == nil {
		return Scope{}, false
	}
	now := time.Now()

	r.registry.mu.RLock()
	if scope, ok := r.registry.hosts[norm]; ok {
		r.registry.mu.RUnlock()
		return scope, true
	}
	if entry, ok := r.registry.cache[norm]; ok && now.Before(entry.expiresAt) {
		r.registry.mu.RUnlock()
		return entry.scope, entry.found
	}
	fallback := r.registry.fallback
	ttl := r.registry.fallbackTTL
	r.registry.mu.RUnlock()

	if fallback == nil {
		return Scope{}, false
	}

	scope, ok := fallback(ctx, norm)
	r.registry.mu.Lock()
	r.registry.cache[norm] = cachedScope{
		scope:     scope,
		found:     ok,
		expiresAt: now.Add(ttl),
	}
	r.registry.mu.Unlock()
	return scope, ok
}

func (r HostResolver) Matches(raw string) bool {
	_, ok := r.Resolve(context.Background(), raw)
	return ok
}

// IsPlatformControlPlanePath returns true for system, webhook, orchestrator, or probe endpoints
// that operate at the platform level rather than within an individual tenant store.
func IsPlatformControlPlanePath(path string) bool {
	return path == "/healthz" ||
		path == "/readyz" ||
		strings.HasPrefix(path, "/api/v1/system/") ||
		strings.HasPrefix(path, "/api/v1/admin/orchestrator") ||
		strings.HasPrefix(path, "/api/v1/superadmin") ||
		strings.HasPrefix(path, "/api/v1/webhooks/") ||
		strings.HasPrefix(path, "/api/v1/internal/")
}

// Middleware resolves the host before downstream auth/repository code. Health,
// readiness, orchestrator, and webhook endpoints remain tenant-neutral or fallback to platform.
func (r HostResolver) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		scope, ok := r.Resolve(req.Context(), req.Host)
		if ok {
			next.ServeHTTP(w, req.WithContext(WithScope(req.Context(), scope)))
			return
		}
		// If host did not resolve to a tenant scope, platform control plane endpoints
		// (e.g. health checks, webhooks, orchestrator, superadmin) should not fail closed with 421.
		if IsPlatformControlPlanePath(req.URL.Path) {
			if platformScope, err := NewScope("platform", "platform"); err == nil {
				next.ServeHTTP(w, req.WithContext(WithScope(req.Context(), platformScope)))
				return
			}
			next.ServeHTTP(w, req)
			return
		}
		if r.require {
			http.Error(w, "tenant unavailable", http.StatusMisdirectedRequest)
			return
		}
		next.ServeHTTP(w, req)
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
