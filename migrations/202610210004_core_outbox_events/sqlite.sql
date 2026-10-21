-- The event outbox belongs to the core: projects without the pages module (the
-- crm preset) get the same table the pages migrations build, in its final
-- shape. Where those migrations already created it, every statement here
-- leaves the existing table and indexes as they are.
CREATE TABLE IF NOT EXISTS outbox_events (
    id TEXT PRIMARY KEY,
    topic TEXT NOT NULL,
    aggregate_id TEXT NOT NULL,
    payload_version INTEGER NOT NULL,
    available_at TEXT NOT NULL DEFAULT '1970-01-01T00:00:00Z',
    attempts INTEGER NOT NULL DEFAULT 0,
    lease_owner TEXT,
    lease_token TEXT,
    lease_until TEXT,
    last_error TEXT,
    tenant_id TEXT NOT NULL DEFAULT ''
);
CREATE INDEX IF NOT EXISTS idx_outbox_claimable ON outbox_events(available_at, lease_until, id);
CREATE INDEX IF NOT EXISTS idx_outbox_tenant_claimable ON outbox_events(tenant_id, available_at, lease_until, id);
