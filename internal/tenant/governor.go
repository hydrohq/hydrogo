package tenant

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type cachedGovernance struct {
	gov       Governance
	expiresAt time.Time
}

type tenantBucket struct {
	mu         sync.Mutex
	tokens     float64
	lastRefill time.Time
	inFlight   atomic.Int64
}

func (b *tenantBucket) allowRPS(rps int, now time.Time) bool {
	if rps <= 0 {
		return true
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	burst := float64(rps * 2)
	if burst < 10 {
		burst = 10
	}
	if b.lastRefill.IsZero() {
		b.tokens = burst - 1
		b.lastRefill = now
		return true
	}
	elapsed := now.Sub(b.lastRefill).Seconds()
	if elapsed > 0 {
		b.tokens += elapsed * float64(rps)
		if b.tokens > burst {
			b.tokens = burst
		}
		b.lastRefill = now
	}
	if b.tokens < 1.0 {
		return false
	}
	b.tokens -= 1.0
	return true
}

// Governor enforces per-tenant lifecycle states (active, readonly, suspended, archived),
// subscription/trial expiration auto-degradation, RPS rate limits, in-flight concurrency
// semaphores, and request body size limits.
type Governor struct {
	db       SQLExecutor
	dialect  string
	cacheTTL time.Duration
	nowFn    func() time.Time

	mu      sync.RWMutex
	cache   map[string]cachedGovernance
	buckets map[string]*tenantBucket
}

// NewGovernor constructs a resource and lifecycle Governor backed by tenant_governance.
func NewGovernor(db SQLExecutor, dialect string) *Governor {
	return &Governor{
		db:       db,
		dialect:  dialect,
		cacheTTL: 5 * time.Second,
		nowFn:    func() time.Time { return time.Now().UTC() },
		cache:    make(map[string]cachedGovernance),
		buckets:  make(map[string]*tenantBucket),
	}
}

// Invalidate evicts a tenant's cached governance state so Orchestrator changes apply immediately.
func (g *Governor) Invalidate(tenantID string) {
	if g == nil {
		return
	}
	tenantID = strings.TrimSpace(tenantID)
	g.mu.Lock()
	delete(g.cache, tenantID)
	g.mu.Unlock()
}

// InvalidateAll clears all cached governance records.
func (g *Governor) InvalidateAll() {
	if g == nil {
		return
	}
	g.mu.Lock()
	g.cache = make(map[string]cachedGovernance)
	g.mu.Unlock()
}

func (g *Governor) loadGovernance(ctx context.Context, tenantID string, now time.Time) Governance {
	g.mu.RLock()
	entry, ok := g.cache[tenantID]
	g.mu.RUnlock()
	if ok && now.Before(entry.expiresAt) {
		entry.gov.EffectiveState = entry.gov.ComputeEffectiveState(now)
		return entry.gov
	}

	gov, err := GetGovernance(ctx, g.db, g.dialect, tenantID)
	if err != nil {
		gov = DefaultGovernance(tenantID)
	}
	gov.EffectiveState = gov.ComputeEffectiveState(now)

	g.mu.Lock()
	g.cache[tenantID] = cachedGovernance{
		gov:       gov,
		expiresAt: now.Add(g.cacheTTL),
	}
	g.mu.Unlock()
	return gov
}

func (g *Governor) bucketFor(tenantID string) *tenantBucket {
	g.mu.RLock()
	b, ok := g.buckets[tenantID]
	g.mu.RUnlock()
	if ok {
		return b
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	if b, ok = g.buckets[tenantID]; ok {
		return b
	}
	b = &tenantBucket{}
	g.buckets[tenantID] = b
	return b
}

// Middleware wraps an HTTP handler with per-tenant lifecycle and resource governance.
func (g *Governor) Middleware(next http.Handler) http.Handler {
	if g == nil {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if isOrchestratorBypassPath(path) {
			next.ServeHTTP(w, r)
			return
		}

		scope, err := FromContext(r.Context())
		if err != nil || strings.TrimSpace(scope.TenantID) == "" {
			next.ServeHTTP(w, r)
			return
		}

		now := g.nowFn()
		gov := g.loadGovernance(r.Context(), scope.TenantID, now)

		switch gov.EffectiveState {
		case LifecycleSuspended, LifecycleArchived:
			writeGovernorError(w, r, http.StatusServiceUnavailable, "workspace_suspended", "This workspace is currently suspended or archived by the platform administrator.")
			return
		case LifecycleReadOnly:
			if isMutatingMethod(r.Method) && !isAuthSessionPath(path) {
				writeGovernorError(w, r, http.StatusPaymentRequired, "workspace_readonly_subscription_expired", "This workspace is in read-only mode because its subscription or trial has expired.")
				return
			}
		}

		bucket := g.bucketFor(scope.TenantID)
		if !bucket.allowRPS(gov.RateLimitRPS, now) {
			writeGovernorError(w, r, http.StatusTooManyRequests, "tenant_rate_limit_exceeded", "Workspace request rate limit exceeded; please slow down.")
			return
		}

		if gov.MaxConcurrentReqs > 0 {
			current := bucket.inFlight.Add(1)
			defer bucket.inFlight.Add(-1)
			if int(current) > gov.MaxConcurrentReqs {
				writeGovernorError(w, r, http.StatusTooManyRequests, "tenant_concurrency_exceeded", "Workspace concurrent request limit exceeded; please try again shortly.")
				return
			}
		}

		if gov.MaxUploadBytes > 0 && r.Body != nil && isMutatingMethod(r.Method) {
			r.Body = http.MaxBytesReader(w, r.Body, gov.MaxUploadBytes)
		}

		next.ServeHTTP(w, r)
	})
}

func isOrchestratorBypassPath(path string) bool {
	return strings.HasPrefix(path, "/api/v1/admin/orchestrator") ||
		strings.HasPrefix(path, "/api/v1/system/") ||
		path == "/healthz" ||
		path == "/readyz"
}

func isAuthSessionPath(path string) bool {
	return strings.HasPrefix(path, "/api/v1/admin/auth/")
}

func isMutatingMethod(method string) bool {
	switch strings.ToUpper(strings.TrimSpace(method)) {
	case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
		return true
	default:
		return false
	}
}

func writeGovernorError(w http.ResponseWriter, r *http.Request, status int, code, message string) {
	reqID := strings.TrimSpace(r.Header.Get("X-Request-Id"))
	if reqID == "" {
		reqID = strings.TrimSpace(w.Header().Get("X-Request-Id"))
	}
	if reqID == "" {
		reqID = "gov-enforced"
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"code":       code,
		"message":    message,
		"request_id": reqID,
	})
}
