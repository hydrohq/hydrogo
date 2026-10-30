package tenant

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"
)

// SQLExecutor encapsulates standard SQL database queries without importing internal/database.
type SQLExecutor interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

const (
	// DefaultMaxStorageBytes is the default storage quota per tenant (5 GiB).
	DefaultMaxStorageBytes int64 = 5 * 1024 * 1024 * 1024
	// DefaultMaxRecordsPerCollection is the default record quota per collection (100,000).
	DefaultMaxRecordsPerCollection int = 100000
)

var (
	ErrStorageQuotaExceeded = errors.New("tenant storage quota exceeded")
	ErrRecordQuotaExceeded  = errors.New("tenant record quota exceeded")
)

// QuotaError provides structured metadata for quota violations.
type QuotaError struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	TenantID  string `json:"tenant_id"`
	Current   int64  `json:"current"`
	Limit     int64  `json:"limit"`
	Requested int64  `json:"requested,omitempty"`
}

func (e *QuotaError) Error() string {
	return e.Message
}

func (e *QuotaError) Is(target error) bool {
	switch target {
	case ErrAIQuotaExceeded:
		return e.Code == "tenant_ai_quota_exceeded"
	case ErrStorageQuotaExceeded:
		return e.Code == "tenant_storage_quota_exceeded"
	case ErrRecordQuotaExceeded:
		return e.Code == "tenant_record_quota_exceeded"
	default:
		return false
	}
}

// Quotas configures maximum consumption bounds for a single tenant.
type Quotas struct {
	MaxStorageBytes          int64 `json:"max_storage_bytes"`
	MaxMonthlyBandwidthBytes int64 `json:"max_monthly_bandwidth_bytes"`
	MaxRecordsPerCollection  int   `json:"max_records_per_collection"`
}

// Usage describes current storage and record metrics for a tenant.
type Usage struct {
	TenantID                string  `json:"tenant_id"`
	StorageBytes            int64   `json:"storage_bytes"`
	MaxStorageBytes         int64   `json:"max_storage_bytes"`
	StorageUsagePercent     float64 `json:"storage_usage_percent"`
	TotalMediaAssets        int64   `json:"total_media_assets"`
	TotalRecords            int64   `json:"total_records"`
	MaxRecordsPerCollection int     `json:"max_records_per_collection"`
}

// GetTenantQuotas loads the configured quotas for a tenant from site_settings or returns defaults.
func GetTenantQuotas(ctx context.Context, db SQLExecutor, dialect string, tenantID string) (Quotas, error) {
	quotas := Quotas{
		MaxStorageBytes:         DefaultMaxStorageBytes,
		MaxRecordsPerCollection: DefaultMaxRecordsPerCollection,
	}

	if db == nil {
		return quotas, nil
	}

	if gov, err := GetGovernance(ctx, db, dialect, tenantID); err == nil {
		if gov.MaxStorageBytes > 0 {
			quotas.MaxStorageBytes = gov.MaxStorageBytes
		}
		if gov.MaxRecords > 0 {
			quotas.MaxRecordsPerCollection = gov.MaxRecords
		}
		return quotas, nil
	}

	p1 := "?"
	if dialect == "postgres" {
		p1 = "$1"
	}

	query := `SELECT key, value FROM site_settings WHERE tenant_id = ` + p1 + ` AND key IN ('tenant.quota.max_storage_bytes', 'tenant.quota.max_monthly_bandwidth_bytes', 'tenant.quota.max_records_per_collection')`
	rows, err := db.QueryContext(ctx, query, tenantID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return quotas, nil
		}
		// In case site_settings does not exist in testing contexts, return defaults
		return quotas, nil
	}
	defer rows.Close()

	for rows.Next() {
		var key, val string
		if err := rows.Scan(&key, &val); err == nil {
			switch key {
			case "tenant.quota.max_storage_bytes":
				if parsed, err := strconv.ParseInt(val, 10, 64); err == nil && parsed > 0 {
					quotas.MaxStorageBytes = parsed
				}
			case "tenant.quota.max_monthly_bandwidth_bytes":
				if parsed, err := strconv.ParseInt(val, 10, 64); err == nil && parsed > 0 {
					quotas.MaxMonthlyBandwidthBytes = parsed
				}
			case "tenant.quota.max_records_per_collection":
				if parsed, err := strconv.Atoi(val); err == nil && parsed > 0 {
					quotas.MaxRecordsPerCollection = parsed
				}
			}
		}
	}

	return quotas, nil
}

// GetTenantUsage computes real-time resource utilization for the given tenant.
func GetTenantUsage(ctx context.Context, db SQLExecutor, dialect string, tenantID string) (Usage, error) {
	quotas, _ := GetTenantQuotas(ctx, db, dialect, tenantID)

	usage := Usage{
		TenantID:                tenantID,
		MaxStorageBytes:         quotas.MaxStorageBytes,
		MaxRecordsPerCollection: quotas.MaxRecordsPerCollection,
	}

	if db == nil {
		return usage, nil
	}

	p1 := "?"
	if dialect == "postgres" {
		p1 = "$1"
	}

	// 1. Media Assets Storage
	var assetBytes int64
	var totalAssets int64
	_ = db.QueryRowContext(ctx, `SELECT COALESCE(SUM(size_bytes), 0), COUNT(*) FROM media_assets WHERE tenant_id = `+p1+` AND deleted_at IS NULL`, tenantID).Scan(&assetBytes, &totalAssets)

	// 2. Media Variants Storage
	var variantBytes int64
	_ = db.QueryRowContext(ctx, `SELECT COALESCE(SUM(size_bytes), 0) FROM media_variants WHERE tenant_id = `+p1, tenantID).Scan(&variantBytes)

	// 3. Collection Records Count
	var totalRecords int64
	_ = db.QueryRowContext(ctx, `SELECT COUNT(*) FROM hydro_collection_records WHERE tenant_id = `+p1, tenantID).Scan(&totalRecords)

	usage.StorageBytes = assetBytes + variantBytes
	usage.TotalMediaAssets = totalAssets
	usage.TotalRecords = totalRecords

	if usage.MaxStorageBytes > 0 {
		usage.StorageUsagePercent = float64(usage.StorageBytes) / float64(usage.MaxStorageBytes) * 100
	}

	return usage, nil
}

// CheckStorageQuota validates that incomingBytes fits within the tenant's remaining storage quota.
func CheckStorageQuota(ctx context.Context, db SQLExecutor, dialect string, tenantID string, incomingBytes int64) error {
	if db == nil || tenantID == "" {
		return nil
	}

	usage, err := GetTenantUsage(ctx, db, dialect, tenantID)
	if err != nil {
		return err
	}

	if usage.MaxStorageBytes > 0 && usage.StorageBytes+incomingBytes > usage.MaxStorageBytes {
		return &QuotaError{
			Code:      "tenant_storage_quota_exceeded",
			Message:   fmt.Sprintf("storage quota exceeded for tenant %s: current %d bytes + upload %d bytes exceeds limit %d bytes", tenantID, usage.StorageBytes, incomingBytes, usage.MaxStorageBytes),
			TenantID:  tenantID,
			Current:   usage.StorageBytes,
			Limit:     usage.MaxStorageBytes,
			Requested: incomingBytes,
		}
	}

	return nil
}

// CheckRecordQuota validates that adding a record does not exceed the collection limit for the tenant.
func CheckRecordQuota(ctx context.Context, db SQLExecutor, dialect string, tenantID, collectionID string) error {
	if db == nil || tenantID == "" || collectionID == "" {
		return nil
	}

	quotas, err := GetTenantQuotas(ctx, db, dialect, tenantID)
	if err != nil {
		return err
	}

	limit := quotas.MaxRecordsPerCollection
	if limit <= 0 {
		return nil
	}

	p1, p2 := "?", "?"
	if dialect == "postgres" {
		p1, p2 = "$1", "$2"
	}

	var count int
	query := `SELECT COUNT(*) FROM hydro_collection_records WHERE tenant_id = ` + p1 + ` AND collection_id = ` + p2
	if err := db.QueryRowContext(ctx, query, tenantID, collectionID).Scan(&count); err != nil {
		return err
	}

	if count >= limit {
		return &QuotaError{
			Code:     "tenant_record_quota_exceeded",
			Message:  fmt.Sprintf("record quota exceeded for collection '%s' in tenant %s: current %d records reaches limit %d", collectionID, tenantID, count, limit),
			TenantID: tenantID,
			Current:  int64(count),
			Limit:    int64(limit),
		}
	}

	return nil
}
