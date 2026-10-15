package auth

import (
	"context"
	"net/http"
)

type PrincipalKind string

const (
	PrincipalSession PrincipalKind = "session"
	PrincipalToken   PrincipalKind = "token"
	PrincipalAPIKey  PrincipalKind = "api_key"
)

// Principal records how the request authenticated. Handlers that need the
// user call UserFromContext; the principal only matters where the credential
// kind changes behaviour (read-only keys, CSRF, token self-service).
type Principal struct {
	Kind    PrincipalKind `json:"kind"`
	Scopes  []string      `json:"scopes,omitempty"`
	TokenID string        `json:"-"`
}

// ReadOnly reports whether the credential may only perform safe requests.
func (p Principal) ReadOnly() bool {
	if p.Kind != PrincipalAPIKey {
		return false
	}
	for _, scope := range p.Scopes {
		if scope == ScopeAdmin {
			return false
		}
	}
	return true
}

// Allows reports whether the principal may use the HTTP method.
func (p Principal) Allows(method string) bool {
	if !p.ReadOnly() {
		return true
	}
	return method == http.MethodGet || method == http.MethodHead || method == http.MethodOptions
}

const principalCtxKey contextKey = "hydro_principal"

func WithPrincipal(ctx context.Context, p Principal) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, principalCtxKey, p)
}

// PrincipalFromContext returns the request principal; an unauthenticated
// request yields the zero value (empty Kind).
func PrincipalFromContext(ctx context.Context) Principal {
	if ctx == nil {
		return Principal{}
	}
	p, _ := ctx.Value(principalCtxKey).(Principal)
	return p
}
