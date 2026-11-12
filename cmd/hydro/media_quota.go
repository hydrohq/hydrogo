package main

import (
	"context"
	"database/sql"

	"github.com/ardaozyildirim/hydrogo/internal/tenant"
)

// mediaQuotaChecker enforces a tenant's storage quota on media uploads.
type mediaQuotaChecker struct {
	db      *sql.DB
	dialect string
}

func (c mediaQuotaChecker) CheckStorageQuota(ctx context.Context, incomingBytes int64) error {
	scope, err := tenant.FromContext(ctx)
	if err != nil {
		return nil
	}
	return tenant.CheckStorageQuota(ctx, c.db, c.dialect, scope.TenantID, incomingBytes)
}
