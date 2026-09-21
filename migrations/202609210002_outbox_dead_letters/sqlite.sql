CREATE TABLE outbox_dead_letters (
    id TEXT PRIMARY KEY,
    tenant_id TEXT NOT NULL DEFAULT '',
    topic TEXT NOT NULL,
    aggregate_id TEXT NOT NULL,
    payload_version INTEGER NOT NULL,
    attempts INTEGER NOT NULL,
    last_error TEXT NOT NULL,
    failed_at TEXT NOT NULL
);
CREATE INDEX idx_outbox_dead_letters_failed_at ON outbox_dead_letters(failed_at, id);
CREATE INDEX idx_outbox_dead_letters_tenant ON outbox_dead_letters(tenant_id, failed_at, id);
