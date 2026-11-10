CREATE TABLE IF NOT EXISTS refund_required (
    id TEXT PRIMARY KEY,
    order_id TEXT NOT NULL,
    provider TEXT NOT NULL,
    amount_cents INTEGER NOT NULL CHECK (amount_cents >= 0),
    currency TEXT NOT NULL,
    reason TEXT NOT NULL,
    created_at TEXT NOT NULL,
    resolved_at TEXT,
    resolved_note TEXT NOT NULL DEFAULT '',
    tenant_id TEXT NOT NULL DEFAULT '',
    UNIQUE (order_id, provider)
);
CREATE INDEX IF NOT EXISTS idx_refund_required_tenant ON refund_required(tenant_id, created_at);

CREATE TABLE IF NOT EXISTS customer_sms_dedup (
    id TEXT PRIMARY KEY,
    order_id TEXT NOT NULL,
    phone TEXT NOT NULL,
    event TEXT NOT NULL,
    dedup_key TEXT NOT NULL UNIQUE,
    sent_at TEXT NOT NULL,
    tenant_id TEXT NOT NULL DEFAULT ''
);
CREATE INDEX IF NOT EXISTS idx_customer_sms_dedup_tenant ON customer_sms_dedup(tenant_id, dedup_key);
