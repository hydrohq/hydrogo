package v1

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

const MaxTenantComponentBytes = 128

var (
	ErrMissingTenantScope = errors.New("missing tenant scope")
	ErrInvalidTenantScope = errors.New("invalid tenant scope")
)

type TenantScope struct {
	TenantID string
	Slug     string
}

func NewTenantScope(tenantID, slug string) (TenantScope, error) {
	if !validTenantComponent(tenantID) || !validTenantComponent(slug) {
		return TenantScope{}, fmt.Errorf("%w: tenant id and slug must be 1-%d ASCII characters", ErrInvalidTenantScope, MaxTenantComponentBytes)
	}
	return TenantScope{TenantID: tenantID, Slug: slug}, nil
}

func WithTenantScope(ctx context.Context, scope TenantScope) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, tenantScopeKey{}, scope)
}

func TenantScopeFromContext(ctx context.Context) (TenantScope, error) {
	if ctx == nil {
		return TenantScope{}, ErrMissingTenantScope
	}
	scope, ok := ctx.Value(tenantScopeKey{}).(TenantScope)
	if !ok || !validTenantComponent(scope.TenantID) || !validTenantComponent(scope.Slug) {
		return TenantScope{}, ErrMissingTenantScope
	}
	return scope, nil
}

func validTenantComponent(value string) bool {
	if value == "" || len(value) > MaxTenantComponentBytes {
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

func TenantCacheKey(ctx context.Context, module, key string) (string, error) {
	scope, err := TenantScopeFromContext(ctx)
	if err != nil {
		return "", err
	}
	if !validTenantComponent(module) || !validTenantComponent(key) {
		return "", ErrInvalidTenantScope
	}
	return strings.Join([]string{"t:" + scope.TenantID, module, key}, ":"), nil
}

type tenantScopeKey struct{}
