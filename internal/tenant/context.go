// Package tenant contains the opt-in tenant-scope primitives used by shared
// multi-tenant adapters. Single-tenant builds do not create a scope.
package tenant

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

var (
	ErrMissingScope = errors.New("security violation: missing tenant scope")
	ErrInvalidScope = errors.New("invalid tenant scope")
)

const maxComponentLength = 128

// Scope identifies the tenant boundary for one request or background job.
// Values are validated before they enter a context or a storage key.
type Scope struct {
	TenantID string
	Slug     string
}

// NewScope validates and constructs an explicit tenant scope.
func NewScope(tenantID, slug string) (Scope, error) {
	if !validComponent(tenantID) || !validComponent(slug) {
		return Scope{}, fmt.Errorf("%w: tenant id and slug must be 1-%d ASCII [A-Za-z0-9._-] characters", ErrInvalidScope, maxComponentLength)
	}
	return Scope{TenantID: tenantID, Slug: slug}, nil
}

// WithScope attaches an already validated scope to ctx.
func WithScope(ctx context.Context, scope Scope) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, scopeKey{}, scope)
}

// FromContext returns the validated scope or fails closed when the caller did
// not explicitly establish one.
func FromContext(ctx context.Context) (Scope, error) {
	if ctx == nil {
		return Scope{}, ErrMissingScope
	}
	scope, ok := ctx.Value(scopeKey{}).(Scope)
	if !ok || !validComponent(scope.TenantID) || !validComponent(scope.Slug) {
		return Scope{}, ErrMissingScope
	}
	return scope, nil
}

// Require is an explicit alias for boundary code that wants fail-closed
// tenant resolution.
func Require(ctx context.Context) (Scope, error) { return FromContext(ctx) }

// CacheKey returns a deterministic tenant-prefixed key. It rejects missing or
// malformed scopes and does not accept arbitrary separators in components.
func CacheKey(ctx context.Context, module, key string) (string, error) {
	scope, err := Require(ctx)
	if err != nil {
		return "", err
	}
	if !validComponent(module) || !validComponent(key) {
		return "", fmt.Errorf("%w: invalid cache key component", ErrInvalidScope)
	}
	return strings.Join([]string{"t:" + scope.TenantID, module, key}, ":"), nil
}

type scopeKey struct{}

func validComponent(value string) bool {
	if value == "" || len(value) > maxComponentLength {
		return false
	}
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '.' || r == '_' || r == '-' {
			continue
		}
		return false
	}
	return true
}
