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
