CREATE TABLE IF NOT EXISTS refund_required (
    id VARCHAR(128) PRIMARY KEY,
    order_id VARCHAR(128) NOT NULL,
    provider VARCHAR(64) NOT NULL,
    amount_cents INTEGER NOT NULL CHECK (amount_cents >= 0),
    currency VARCHAR(8) NOT NULL,
    reason TEXT NOT NULL,
    created_at DATETIME(6) NOT NULL,
    resolved_at DATETIME(6) NULL,
    resolved_note TEXT NOT NULL,
    tenant_id VARCHAR(128) NOT NULL DEFAULT '',
    UNIQUE KEY idx_refund_required_order_provider (order_id, provider),
    INDEX idx_refund_required_tenant (tenant_id, created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS customer_sms_dedup (
    id VARCHAR(128) PRIMARY KEY,
    order_id VARCHAR(128) NOT NULL,
    phone VARCHAR(64) NOT NULL,
    event VARCHAR(64) NOT NULL,
    dedup_key VARCHAR(191) NOT NULL,
    sent_at DATETIME(6) NOT NULL,
    tenant_id VARCHAR(128) NOT NULL DEFAULT '',
    UNIQUE KEY idx_customer_sms_dedup_key (dedup_key),
    INDEX idx_customer_sms_dedup_tenant (tenant_id, dedup_key)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
