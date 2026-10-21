-- The event outbox belongs to the core: projects without the pages module (the
-- crm preset) get the same table the pages migrations build, in its final
-- shape. Where those migrations already created it, the table and its indexes
-- stay as they are.
CREATE TABLE IF NOT EXISTS outbox_events (
    id VARCHAR(128) PRIMARY KEY,
    topic VARCHAR(128) NOT NULL,
    aggregate_id VARCHAR(128) NOT NULL,
    payload_version INTEGER NOT NULL,
    available_at DATETIME(6) NOT NULL DEFAULT '1970-01-01 00:00:00.000000',
    attempts INTEGER NOT NULL DEFAULT 0,
    lease_owner VARCHAR(128),
    lease_token VARCHAR(128),
    lease_until DATETIME(6),
    last_error LONGTEXT,
    tenant_id VARCHAR(128) NOT NULL DEFAULT '',
    INDEX idx_outbox_claimable (available_at, lease_until, id),
    INDEX idx_outbox_tenant_claimable (tenant_id, available_at, lease_until, id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
