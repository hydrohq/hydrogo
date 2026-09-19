package media

import (
	"context"
	"errors"
	"path/filepath"
	"strings"

	"github.com/ardaozyildirim/hydrogo/internal/tenant"
)

var ErrInvalidStorageKey = errors.New("invalid media storage key")

// StorageKeyForContext gives shared deployments a tenant-owned object prefix.
// A context without a tenant retains the legacy single-customer basename so
// existing public media and local filesystem deployments remain compatible.
func StorageKeyForContext(ctx context.Context, key string) (string, error) {
	base := filepath.Base(strings.TrimSpace(key))
	if base == "" || base == "." || base == string(filepath.Separator) || strings.ContainsAny(base, "\\/\x00") {
		return "", ErrInvalidStorageKey
	}
	scope, err := tenant.FromContext(ctx)
	if err != nil {
		return base, nil
	}
	return "tenants/" + scope.TenantID + "/" + base, nil
}

// ContextForStorageKey restores the tenant boundary for a queued derivative
// job when the job carries only the immutable object key. Legacy keys remain
// unscoped for single-customer compatibility.
func ContextForStorageKey(ctx context.Context, key string) (context.Context, error) {
	parts := strings.Split(strings.TrimSpace(key), "/")
	if len(parts) < 3 || parts[0] != "tenants" || parts[1] == "" {
		return ctx, nil
	}
	scope, err := tenant.NewScope(parts[1], parts[1])
	if err != nil {
		return nil, err
	}
	return tenant.WithScope(ctx, scope), nil
}
