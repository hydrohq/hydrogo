CREATE TABLE IF NOT EXISTS refund_required (
    id TEXT PRIMARY KEY,
    order_id TEXT NOT NULL,
    provider TEXT NOT NULL,
    amount_cents INTEGER NOT NULL CHECK (amount_cents >= 0),
    currency TEXT NOT NULL,
    reason TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    resolved_at TIMESTAMPTZ,
    resolved_note TEXT NOT NULL DEFAULT '',
    tenant_id TEXT NOT NULL DEFAULT '',
    UNIQUE (order_id, provider)
);
CREATE INDEX IF NOT EXISTS idx_refund_required_tenant ON refund_required(tenant_id, created_at);

ALTER TABLE refund_required ENABLE ROW LEVEL SECURITY;
ALTER TABLE refund_required FORCE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS hydrogo_tenant_isolation ON refund_required;
CREATE POLICY hydrogo_tenant_isolation ON refund_required
    USING ((coalesce(nullif(current_setting('app.tenant_rls_enforced', true), ''), 'false') <> 'true') OR tenant_id = nullif(current_setting('app.current_tenant_id', true), ''))
    WITH CHECK ((coalesce(nullif(current_setting('app.tenant_rls_enforced', true), ''), 'false') <> 'true') OR tenant_id = nullif(current_setting('app.current_tenant_id', true), ''));

CREATE TABLE IF NOT EXISTS customer_sms_dedup (
    id TEXT PRIMARY KEY,
    order_id TEXT NOT NULL,
    phone TEXT NOT NULL,
    event TEXT NOT NULL,
    dedup_key TEXT NOT NULL UNIQUE,
    sent_at TIMESTAMPTZ NOT NULL,
    tenant_id TEXT NOT NULL DEFAULT ''
);
CREATE INDEX IF NOT EXISTS idx_customer_sms_dedup_tenant ON customer_sms_dedup(tenant_id, dedup_key);

ALTER TABLE customer_sms_dedup ENABLE ROW LEVEL SECURITY;
ALTER TABLE customer_sms_dedup FORCE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS hydrogo_tenant_isolation ON customer_sms_dedup;
CREATE POLICY hydrogo_tenant_isolation ON customer_sms_dedup
    USING ((coalesce(nullif(current_setting('app.tenant_rls_enforced', true), ''), 'false') <> 'true') OR tenant_id = nullif(current_setting('app.current_tenant_id', true), ''))
    WITH CHECK ((coalesce(nullif(current_setting('app.tenant_rls_enforced', true), ''), 'false') <> 'true') OR tenant_id = nullif(current_setting('app.current_tenant_id', true), ''));
