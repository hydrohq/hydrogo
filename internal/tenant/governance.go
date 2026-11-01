package tenant

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

const (
	LifecycleProvisioning = "provisioning"
	LifecycleActive       = "active"
	LifecycleDemo         = "demo"
	LifecycleReadOnly     = "readonly"
	LifecycleSuspended    = "suspended"
	LifecycleArchived     = "archived"
)

const (
	PlanStarter = "starter"
	PlanPro     = "pro"
	PlanVIP     = "vip"
	PlanCustom  = "custom"
)

var (
	ErrGovernanceNotFound = errors.New("tenant governance not found")
	ErrAIQuotaExceeded    = errors.New("tenant monthly AI token quota exceeded")
	ErrAIDisabled         = errors.New("tenant AI assistant is disabled")
)

// Governance holds the platform orchestrator's resource, lifecycle, domain, and AI controls for a tenant workspace.
type Governance struct {
	TenantID            string `json:"tenant_id"`
	DisplayName         string `json:"display_name"`
	Subdomain           string `json:"subdomain"`
	CustomDomain        string `json:"custom_domain"`
	DNSVerified         bool   `json:"dns_verified"`
	Slug                string `json:"slug,omitempty"`
	Status              string `json:"status,omitempty"`
	Blueprint           string `json:"blueprint"`
	Plan                string `json:"plan"`
	LifecycleState      string `json:"lifecycle_state"`
	EffectiveState      string `json:"effective_state"`
	TrialEndsAt         string `json:"trial_ends_at,omitempty"`
	SubscriptionEndsAt  string `json:"subscription_ends_at,omitempty"`
	MaxStorageBytes     int64  `json:"max_storage_bytes"`
	MaxRecords          int    `json:"max_records"`
	RateLimitRPS        int    `json:"rate_limit_rps"`
	MaxConcurrentReqs   int    `json:"max_concurrent_reqs"`
	MaxUploadBytes      int64  `json:"max_upload_bytes"`
	AIEnabled           bool   `json:"ai_enabled"`
	AIProvider          string `json:"ai_provider"`
	AIModel             string `json:"ai_model"`
	AIEncryptedAPIKey   string `json:"-"`
	HasCustomAPIKey     bool   `json:"has_custom_api_key"`
	AIMonthlyTokenLimit int64  `json:"ai_monthly_token_limit"`
	AITokensUsed        int64  `json:"ai_tokens_used"`
	// AITokenPeriod is the "YYYY-MM" bucket AITokensUsed belongs to; crossing
	// into a new calendar month resets the counter instead of requiring a job.
	AITokenPeriod string    `json:"ai_token_period"`
	OwnerPhone    string    `json:"owner_phone"`
	Notes         string    `json:"notes"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// WorkspaceSummary combines Governance controls with live storage, record, and AI utilization metrics.
type WorkspaceSummary struct {
	Governance          Governance `json:"governance"`
	Usage               Usage      `json:"usage"`
	AdminEmail          string     `json:"admin_email,omitempty"`
	RecordUsagePercent  float64    `json:"record_usage_percent"`
	AITokenUsagePercent float64    `json:"ai_token_usage_percent"`
}

// DefaultGovernance returns the baseline starter governance settings for a tenant.
func DefaultGovernance(tenantID string) Governance {
	now := time.Now().UTC()
	gov := Governance{
		TenantID:       strings.TrimSpace(tenantID),
		DisplayName:    strings.TrimSpace(tenantID),
		Subdomain:      strings.TrimSpace(tenantID),
		Blueprint:      "boutique",
		Plan:           PlanStarter,
		LifecycleState: LifecycleActive,
		EffectiveState: LifecycleActive,
		AIEnabled:      true,
		AIProvider:     "gemini",
		AIModel:        "gemini-2.5-flash",
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	ApplyPlanPreset(&gov, PlanStarter)
	return gov
}

// ApplyPlanPreset sets resource quotas according to a named subscription tier.
func ApplyPlanPreset(gov *Governance, plan string) {
	if gov == nil {
		return
	}
	switch strings.ToLower(strings.TrimSpace(plan)) {
	case PlanVIP:
		gov.Plan = PlanVIP
		gov.MaxStorageBytes = 10 * 1024 * 1024 * 1024 // 10 GiB
		gov.MaxRecords = 100000
		gov.RateLimitRPS = 120
		gov.MaxConcurrentReqs = 60
		gov.MaxUploadBytes = 50 * 1024 * 1024 // 50 MiB
		gov.AIMonthlyTokenLimit = 10000000
	case PlanPro:
		gov.Plan = PlanPro
		gov.MaxStorageBytes = 2 * 1024 * 1024 * 1024 // 2 GiB
		gov.MaxRecords = 25000
		gov.RateLimitRPS = 60
		gov.MaxConcurrentReqs = 30
		gov.MaxUploadBytes = 25 * 1024 * 1024 // 25 MiB
		gov.AIMonthlyTokenLimit = 2000000
	case PlanCustom:
		gov.Plan = PlanCustom
	default:
		gov.Plan = PlanStarter
		gov.MaxStorageBytes = 512 * 1024 * 1024 // 512 MiB
		gov.MaxRecords = 5000
		gov.RateLimitRPS = 30
		gov.MaxConcurrentReqs = 15
		gov.MaxUploadBytes = 10 * 1024 * 1024 // 10 MiB
		gov.AIMonthlyTokenLimit = 500000
	}
}

// ComputeEffectiveState evaluates the configured lifecycle state along with trial/subscription expiry timestamps.
// If an active workspace's subscription or trial has expired, it automatically degrades to "readonly" so the public
// storefront stays online while admin/AI mutations are frozen until renewal.
func (g Governance) ComputeEffectiveState(now time.Time) string {
	state := strings.ToLower(strings.TrimSpace(g.LifecycleState))
	if state == "" {
		state = LifecycleActive
	}
	if state != LifecycleActive && state != LifecycleDemo {
		return state
	}
	if raw := strings.TrimSpace(g.SubscriptionEndsAt); raw != "" {
		if exp, err := time.Parse(time.RFC3339, raw); err == nil && now.After(exp) {
			return LifecycleReadOnly
		}
	}
	if raw := strings.TrimSpace(g.TrialEndsAt); raw != "" && strings.TrimSpace(g.SubscriptionEndsAt) == "" {
		if exp, err := time.Parse(time.RFC3339, raw); err == nil && now.After(exp) {
			return LifecycleReadOnly
		}
	}
	return state
}

func ValidLifecycleState(state string) bool {
	switch strings.ToLower(strings.TrimSpace(state)) {
	case LifecycleProvisioning, LifecycleActive, LifecycleDemo, LifecycleReadOnly, LifecycleSuspended, LifecycleArchived:
		return true
	default:
		return false
	}
}

// UpsertGovernance creates or updates a tenant's governance record.
func UpsertGovernance(ctx context.Context, db SQLExecutor, dialect string, gov Governance) error {
	if db == nil {
		return errors.New("database is required")
	}
	gov.TenantID = strings.TrimSpace(gov.TenantID)
	if _, err := NewScope(gov.TenantID, gov.TenantID); err != nil {
		return err
	}
	if !ValidLifecycleState(gov.LifecycleState) {
		gov.LifecycleState = LifecycleActive
	}
	if strings.TrimSpace(gov.Blueprint) == "" {
		gov.Blueprint = "boutique"
	}
	if strings.TrimSpace(gov.Plan) == "" {
		gov.Plan = PlanStarter
	}
	if strings.TrimSpace(gov.AIProvider) == "" {
		gov.AIProvider = "gemini"
	}
	if strings.TrimSpace(gov.AIModel) == "" {
		gov.AIModel = "gemini-2.5-flash"
	}
	now := time.Now().UTC()
	if gov.CreatedAt.IsZero() {
		gov.CreatedAt = now
	}
	gov.UpdatedAt = now
	aiEnabledInt := 0
	if gov.AIEnabled {
		aiEnabledInt = 1
	}
	dnsVerifiedVal := any(0)
	if dialect == "postgres" {
		dnsVerifiedVal = gov.DNSVerified
	} else if gov.DNSVerified {
		dnsVerifiedVal = 1
	}

	cols := `tenant_id, display_name, subdomain, custom_domain, blueprint, plan, lifecycle_state,
		trial_ends_at, subscription_ends_at, max_storage_bytes, max_records, rate_limit_rps,
		max_concurrent_reqs, max_upload_bytes, ai_enabled, ai_provider, ai_model,
		ai_encrypted_api_key, ai_monthly_token_limit, ai_tokens_used, ai_token_period, owner_phone, notes,
		created_at, updated_at, dns_verified`

	var ph []string
	for i := 1; i <= 26; i++ {
		ph = append(ph, placeholder(dialect, i))
	}
	valuesClause := strings.Join(ph, ", ")

	var query string
	if dialect == "mysql" {
		query = `INSERT INTO tenant_governance (` + cols + `) VALUES (` + valuesClause + `)
			ON DUPLICATE KEY UPDATE
				display_name = VALUES(display_name),
				subdomain = VALUES(subdomain),
				custom_domain = VALUES(custom_domain),
				blueprint = VALUES(blueprint),
				plan = VALUES(plan),
				lifecycle_state = VALUES(lifecycle_state),
				trial_ends_at = VALUES(trial_ends_at),
				subscription_ends_at = VALUES(subscription_ends_at),
				max_storage_bytes = VALUES(max_storage_bytes),
				max_records = VALUES(max_records),
				rate_limit_rps = VALUES(rate_limit_rps),
				max_concurrent_reqs = VALUES(max_concurrent_reqs),
				max_upload_bytes = VALUES(max_upload_bytes),
				ai_enabled = VALUES(ai_enabled),
				ai_provider = VALUES(ai_provider),
				ai_model = VALUES(ai_model),
				ai_encrypted_api_key = VALUES(ai_encrypted_api_key),
				ai_monthly_token_limit = VALUES(ai_monthly_token_limit),
				ai_tokens_used = VALUES(ai_tokens_used),
				ai_token_period = VALUES(ai_token_period),
				owner_phone = VALUES(owner_phone),
				notes = VALUES(notes),
				updated_at = VALUES(updated_at),
				dns_verified = VALUES(dns_verified)`
	} else {
		query = `INSERT INTO tenant_governance (` + cols + `) VALUES (` + valuesClause + `)
			ON CONFLICT(tenant_id) DO UPDATE SET
				display_name = excluded.display_name,
				subdomain = excluded.subdomain,
				custom_domain = excluded.custom_domain,
				blueprint = excluded.blueprint,
				plan = excluded.plan,
				lifecycle_state = excluded.lifecycle_state,
				trial_ends_at = excluded.trial_ends_at,
				subscription_ends_at = excluded.subscription_ends_at,
				max_storage_bytes = excluded.max_storage_bytes,
				max_records = excluded.max_records,
				rate_limit_rps = excluded.rate_limit_rps,
				max_concurrent_reqs = excluded.max_concurrent_reqs,
				max_upload_bytes = excluded.max_upload_bytes,
				ai_enabled = excluded.ai_enabled,
				ai_provider = excluded.ai_provider,
				ai_model = excluded.ai_model,
				ai_encrypted_api_key = excluded.ai_encrypted_api_key,
				ai_monthly_token_limit = excluded.ai_monthly_token_limit,
				ai_tokens_used = excluded.ai_tokens_used,
				ai_token_period = excluded.ai_token_period,
				owner_phone = excluded.owner_phone,
				notes = excluded.notes,
				updated_at = excluded.updated_at,
				dns_verified = excluded.dns_verified`
	}

	_, err := db.ExecContext(ctx, query,
		gov.TenantID,
		strings.TrimSpace(gov.DisplayName),
		strings.ToLower(strings.TrimSpace(gov.Subdomain)),
		strings.ToLower(strings.TrimSpace(gov.CustomDomain)),
		strings.TrimSpace(gov.Blueprint),
		strings.TrimSpace(gov.Plan),
		strings.ToLower(strings.TrimSpace(gov.LifecycleState)),
		strings.TrimSpace(gov.TrialEndsAt),
		strings.TrimSpace(gov.SubscriptionEndsAt),
		gov.MaxStorageBytes,
		gov.MaxRecords,
		gov.RateLimitRPS,
		gov.MaxConcurrentReqs,
		gov.MaxUploadBytes,
		aiEnabledInt,
		strings.TrimSpace(gov.AIProvider),
		strings.TrimSpace(gov.AIModel),
		gov.AIEncryptedAPIKey,
		gov.AIMonthlyTokenLimit,
		gov.AITokensUsed,
		strings.TrimSpace(gov.AITokenPeriod),
		strings.TrimSpace(gov.OwnerPhone),
		gov.Notes,
		dbTime(dialect, gov.CreatedAt),
		dbTime(dialect, gov.UpdatedAt),
		dnsVerifiedVal,
	)
	return err
}

const selectGovernanceColumns = `tenant_id, display_name, subdomain, custom_domain, blueprint, plan, lifecycle_state,
	trial_ends_at, subscription_ends_at, max_storage_bytes, max_records, rate_limit_rps,
	max_concurrent_reqs, max_upload_bytes, ai_enabled, ai_provider, ai_model,
	COALESCE(ai_encrypted_api_key, ''), ai_monthly_token_limit, ai_tokens_used,
	COALESCE(ai_token_period, ''), owner_phone, COALESCE(notes, ''), created_at, updated_at, COALESCE(dns_verified, 0)`

type rowScanner interface {
	Scan(dest ...any) error
}

func scanGovernance(scanner rowScanner, now time.Time) (Governance, error) {
	var gov Governance
	var aiEnabledInt int
	var createdRaw, updatedRaw, dnsVerifiedRaw any
	err := scanner.Scan(
		&gov.TenantID,
		&gov.DisplayName,
		&gov.Subdomain,
		&gov.CustomDomain,
		&gov.Blueprint,
		&gov.Plan,
		&gov.LifecycleState,
		&gov.TrialEndsAt,
		&gov.SubscriptionEndsAt,
		&gov.MaxStorageBytes,
		&gov.MaxRecords,
		&gov.RateLimitRPS,
		&gov.MaxConcurrentReqs,
		&gov.MaxUploadBytes,
		&aiEnabledInt,
		&gov.AIProvider,
		&gov.AIModel,
		&gov.AIEncryptedAPIKey,
		&gov.AIMonthlyTokenLimit,
		&gov.AITokensUsed,
		&gov.AITokenPeriod,
		&gov.OwnerPhone,
		&gov.Notes,
		&createdRaw,
		&updatedRaw,
		&dnsVerifiedRaw,
	)
	if err != nil {
		return Governance{}, err
	}
	gov.AIEnabled = aiEnabledInt != 0
	switch v := dnsVerifiedRaw.(type) {
	case bool:
		gov.DNSVerified = v
	case int64:
		gov.DNSVerified = v != 0
	case int:
		gov.DNSVerified = v != 0
	case []byte:
		gov.DNSVerified = len(v) > 0 && v[0] != '0' && v[0] != 0
	}
	if gov.Slug == "" {
		gov.Slug = gov.Subdomain
		if gov.Slug == "" {
			gov.Slug = gov.TenantID
		}
	}
	gov.Status = gov.LifecycleState
	gov.HasCustomAPIKey = strings.TrimSpace(gov.AIEncryptedAPIKey) != ""
	gov.CreatedAt = parseDBTime(createdRaw)
	gov.UpdatedAt = parseDBTime(updatedRaw)
	gov.EffectiveState = gov.ComputeEffectiveState(now)
	return gov, nil
}

func parseDBTime(raw any) time.Time {
	switch v := raw.(type) {
	case time.Time:
		return v.UTC()
	case string:
		for _, layout := range []string{time.RFC3339Nano, time.RFC3339, "2006-01-02 15:04:05.999999", "2006-01-02 15:04:05"} {
			if parsed, err := time.Parse(layout, v); err == nil {
				return parsed.UTC()
			}
		}
	case []byte:
		return parseDBTime(string(v))
	}
	return time.Time{}
}

// GetGovernance loads the governance configuration for a single tenant, returning DefaultGovernance if none is stored yet.
func GetGovernance(ctx context.Context, db SQLExecutor, dialect, tenantID string) (Governance, error) {
	tenantID = strings.TrimSpace(tenantID)
	if db == nil || tenantID == "" {
		return DefaultGovernance(tenantID), nil
	}
	query := `SELECT ` + selectGovernanceColumns + ` FROM tenant_governance WHERE tenant_id = ` + placeholder(dialect, 1)
	gov, err := scanGovernance(db.QueryRowContext(ctx, query, tenantID), time.Now().UTC())
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return DefaultGovernance(tenantID), ErrGovernanceNotFound
		}
		return DefaultGovernance(tenantID), err
	}
	return gov, nil
}

// ListGovernances returns all tenants from the registry merged with their governance configuration.
func ListGovernances(ctx context.Context, db SQLExecutor, dialect string) ([]Governance, error) {
	if db == nil {
		return nil, errors.New("database is required")
	}
	now := time.Now().UTC()
	rows, err := db.QueryContext(ctx, `SELECT `+selectGovernanceColumns+` FROM tenant_governance ORDER BY created_at DESC, tenant_id ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	byTenant := make(map[string]Governance)
	var list []Governance
	for rows.Next() {
		gov, err := scanGovernance(rows, now)
		if err != nil {
			return nil, err
		}
		byTenant[gov.TenantID] = gov
		list = append(list, gov)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Also include any tenants in the base `tenants` registry that do not yet have a `tenant_governance` row.
	regRows, err := db.QueryContext(ctx, `SELECT tenant_id, state, created_at, updated_at FROM tenants ORDER BY tenant_id ASC`)
	if err == nil {
		defer regRows.Close()
		for regRows.Next() {
			var tid, state string
			var cRaw, uRaw any
			if err := regRows.Scan(&tid, &state, &cRaw, &uRaw); err == nil {
				if _, exists := byTenant[tid]; !exists {
					gov := DefaultGovernance(tid)
					if ValidLifecycleState(state) {
						gov.LifecycleState = state
						gov.EffectiveState = state
					}
					gov.CreatedAt = parseDBTime(cRaw)
					gov.UpdatedAt = parseDBTime(uRaw)
					list = append(list, gov)
				}
			}
		}
	}
	return list, nil
}

// ResolveTenantByDomain looks up an active/non-archived tenant matching the given host by custom_domain or subdomain.
func ResolveTenantByDomain(ctx context.Context, db SQLExecutor, dialect, host string) (Scope, bool) {
	host = normalizeHost(host)
	if db == nil || host == "" {
		return Scope{}, false
	}
	sub := host
	if idx := strings.IndexByte(host, '.'); idx > 0 {
		sub = host[:idx]
	}
	p1, p2, p3, p4 := placeholder(dialect, 1), placeholder(dialect, 2), placeholder(dialect, 3), placeholder(dialect, 4)
	query := `SELECT tenant_id FROM tenant_governance
		WHERE lifecycle_state <> 'archived'
		  AND (custom_domain = ` + p1 + ` OR subdomain = ` + p2 + ` OR subdomain = ` + p3 + ` OR tenant_id = ` + p4 + `)
		LIMIT 1`
	var tenantID string
	if err := db.QueryRowContext(ctx, query, host, host, sub, sub).Scan(&tenantID); err != nil || strings.TrimSpace(tenantID) == "" {
		return Scope{}, false
	}
	scope, err := NewScope(tenantID, tenantID)
	if err != nil {
		return Scope{}, false
	}
	return scope, true
}

// CheckDomainAllowed verifies whether Caddy On-Demand TLS is permitted to issue a certificate for domain.
func CheckDomainAllowed(ctx context.Context, db SQLExecutor, dialect, domain string) (bool, error) {
	domain = normalizeHost(domain)
	if db == nil || domain == "" || strings.Contains(domain, "..") {
		return false, nil
	}
	sub := ""
	if idx := strings.IndexByte(domain, '.'); idx > 0 {
		sub = domain[:idx]
	}
	p1, p2, p3, p4 := placeholder(dialect, 1), placeholder(dialect, 2), placeholder(dialect, 3), placeholder(dialect, 4)
	query := `SELECT COUNT(*) FROM tenant_governance
		WHERE lifecycle_state NOT IN ('suspended', 'archived')
		  AND ((custom_domain <> '' AND custom_domain = ` + p1 + `) OR (subdomain <> '' AND (subdomain = ` + p2 + ` OR subdomain = ` + p3 + `)) OR tenant_id = ` + p4 + `)`
	var count int
	if err := db.QueryRowContext(ctx, query, domain, domain, sub, sub).Scan(&count); err != nil {
		return false, err
	}
	return count > 0, nil
}

// ConsumeAITokens checks if the tenant has AI enabled and remaining monthly token quota, then increments ai_tokens_used.
func ConsumeAITokens(ctx context.Context, db SQLExecutor, dialect, tenantID string, tokens int64) (Governance, error) {
	gov, err := GetGovernance(ctx, db, dialect, tenantID)
	if err != nil && !errors.Is(err, ErrGovernanceNotFound) {
		return gov, err
	}
	if errors.Is(err, ErrGovernanceNotFound) {
		if upsertErr := UpsertGovernance(ctx, db, dialect, gov); upsertErr != nil {
			return gov, upsertErr
		}
	}
	if !gov.AIEnabled {
		return gov, ErrAIDisabled
	}
	if effective := gov.ComputeEffectiveState(time.Now().UTC()); effective != LifecycleActive && effective != LifecycleDemo {
		return gov, fmt.Errorf("workspace is in %s state", effective)
	}
	if tokens < 0 {
		tokens = 0
	}
	// Crossed into a new calendar month: previous usage no longer counts. The
	// reset is folded into consumption so no scheduler is required and the
	// counter self-heals on first use after the rollover.
	now := time.Now().UTC()
	period := now.Format("2006-01")
	if gov.AITokenPeriod != period {
		p1, p2, p3 := placeholder(dialect, 1), placeholder(dialect, 2), placeholder(dialect, 3)
		resetQuery := `UPDATE tenant_governance SET ai_tokens_used = 0, ai_token_period = ` + p1 + `, updated_at = ` + p2 + ` WHERE tenant_id = ` + p3
		if _, err := db.ExecContext(ctx, resetQuery, period, dbTime(dialect, now), tenantID); err != nil {
			return gov, err
		}
		gov.AITokensUsed = 0
		gov.AITokenPeriod = period
	}
	if gov.AIMonthlyTokenLimit > 0 && ((tokens == 0 && gov.AITokensUsed >= gov.AIMonthlyTokenLimit) || gov.AITokensUsed+tokens > gov.AIMonthlyTokenLimit) {
		return gov, &QuotaError{
			Code:      "tenant_ai_quota_exceeded",
			Message:   fmt.Sprintf("monthly AI token quota exceeded for tenant %s: %d + %d > %d", tenantID, gov.AITokensUsed, tokens, gov.AIMonthlyTokenLimit),
			TenantID:  tenantID,
			Current:   gov.AITokensUsed,
			Limit:     gov.AIMonthlyTokenLimit,
			Requested: tokens,
		}
	}
	if tokens > 0 {
		p1, p2, p3 := placeholder(dialect, 1), placeholder(dialect, 2), placeholder(dialect, 3)
		query := `UPDATE tenant_governance SET ai_tokens_used = ai_tokens_used + ` + p1 + `, updated_at = ` + p2 + ` WHERE tenant_id = ` + p3
		if _, err := db.ExecContext(ctx, query, tokens, dbTime(dialect, now), tenantID); err != nil {
			return gov, err
		}
		gov.AITokensUsed += tokens
	}
	return gov, nil
}
