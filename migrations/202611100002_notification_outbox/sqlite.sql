CREATE TABLE IF NOT EXISTS notification_outbox (
    id TEXT PRIMARY KEY,
    channel TEXT NOT NULL,
    payload TEXT NOT NULL,
    attempts INTEGER NOT NULL DEFAULT 0,
    next_attempt_at TIMESTAMP NOT NULL,
    sent_at TIMESTAMP,
    last_error TEXT NOT NULL DEFAULT '',
    tenant_id TEXT NOT NULL DEFAULT ''
);
CREATE INDEX IF NOT EXISTS idx_notification_outbox_poll ON notification_outbox(sent_at, next_attempt_at);
CREATE INDEX IF NOT EXISTS idx_notification_outbox_tenant ON notification_outbox(tenant_id, sent_at, next_attempt_at);
