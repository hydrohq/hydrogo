CREATE TABLE IF NOT EXISTS tenant_governance (
    tenant_id TEXT PRIMARY KEY,
    display_name TEXT NOT NULL DEFAULT '',
    subdomain TEXT NOT NULL DEFAULT '',
    custom_domain TEXT NOT NULL DEFAULT '',
    blueprint TEXT NOT NULL DEFAULT 'boutique',
    plan TEXT NOT NULL DEFAULT 'starter',
    lifecycle_state TEXT NOT NULL DEFAULT 'active' CHECK (lifecycle_state IN ('provisioning', 'active', 'readonly', 'suspended', 'archived')),
    trial_ends_at TEXT NOT NULL DEFAULT '',
    subscription_ends_at TEXT NOT NULL DEFAULT '',
    max_storage_bytes BIGINT NOT NULL DEFAULT 536870912,
    max_records INTEGER NOT NULL DEFAULT 5000,
    rate_limit_rps INTEGER NOT NULL DEFAULT 30,
    max_concurrent_reqs INTEGER NOT NULL DEFAULT 15,
    max_upload_bytes BIGINT NOT NULL DEFAULT 10485760,
    ai_enabled INTEGER NOT NULL DEFAULT 1,
    ai_provider TEXT NOT NULL DEFAULT 'gemini',
    ai_model TEXT NOT NULL DEFAULT 'gemini-2.5-flash',
    ai_encrypted_api_key TEXT NOT NULL DEFAULT '',
    ai_monthly_token_limit BIGINT NOT NULL DEFAULT 500000,
    ai_tokens_used BIGINT NOT NULL DEFAULT 0,
    owner_phone TEXT NOT NULL DEFAULT '',
    notes TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_tenant_governance_subdomain ON tenant_governance(subdomain);
CREATE INDEX IF NOT EXISTS idx_tenant_governance_custom_domain ON tenant_governance(custom_domain);
