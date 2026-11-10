CREATE TABLE IF NOT EXISTS notification_outbox (
    id VARCHAR(64) PRIMARY KEY,
    channel VARCHAR(64) NOT NULL,
    payload TEXT NOT NULL,
    attempts INT NOT NULL DEFAULT 0,
    next_attempt_at TIMESTAMP NOT NULL,
    sent_at TIMESTAMP NULL,
    last_error TEXT NOT NULL,
    tenant_id VARCHAR(64) NOT NULL DEFAULT '',
    INDEX idx_notification_outbox_poll (sent_at, next_attempt_at),
    INDEX idx_notification_outbox_tenant (tenant_id, sent_at, next_attempt_at)
);
