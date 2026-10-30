package audit

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"

	auditstore "github.com/ardaozyildirim/hydrogo/internal/audit/sqlstore"
	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/internal/tenant"
)

// VerificationResult describes the cryptographic verification outcome of a tenant's audit chain.
type VerificationResult struct {
	Valid        bool     `json:"valid"`
	TenantID     string   `json:"tenant_id"`
	TotalLinks   int64    `json:"total_links"`
	LegacyEvents int64    `json:"legacy_events"`
	HeadSeq      int64    `json:"head_seq"`
	HeadHash     string   `json:"head_hash"`
	Problems     []string `json:"problems,omitempty"`
}

// Service provides high-level audit operations including cryptographic chain verification.
type Service struct {
	Store              auditstore.Store
	DB                 *sql.DB
	Dialect            database.Dialect
	RequireTenantScope bool
}

// VerifyAuditChain recomputes SHA-256 hashes from genesis to head for the given tenant,
// confirming that historical records have not been inserted, mutated, or deleted.
func (s Service) VerifyAuditChain(ctx context.Context, tenantID string) (VerificationResult, error) {
	if s.RequireTenantScope && tenantID == "" {
		if scope, err := tenant.FromContext(ctx); err == nil {
			tenantID = scope.TenantID
		} else {
			return VerificationResult{}, tenant.ErrMissingScope
		}
	} else if tenantID == "" {
		if scope, err := tenant.FromContext(ctx); err == nil {
			tenantID = scope.TenantID
		}
	}

	report, err := s.Store.VerifyChain(ctx, s.DB, tenantID, nil)
	if err != nil {
		return VerificationResult{}, fmt.Errorf("audit chain verification failed: %w", err)
	}

	valid := len(report.Problems) == 0
	if !valid {
		slog.Error("audit chain integrity violation detected", "tenant_id", tenantID, "problems", report.Problems)
	}

	return VerificationResult{
		Valid:        valid,
		TenantID:     report.TenantID,
		TotalLinks:   report.Links,
		LegacyEvents: report.Legacy,
		HeadSeq:      report.Head.Seq,
		HeadHash:     report.Head.Hash,
		Problems:     report.Problems,
	}, nil
}
