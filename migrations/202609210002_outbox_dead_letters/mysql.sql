CREATE TABLE outbox_dead_letters (
    id VARCHAR(128) PRIMARY KEY,
    tenant_id VARCHAR(128) NOT NULL DEFAULT '',
    topic VARCHAR(128) NOT NULL,
    aggregate_id VARCHAR(128) NOT NULL,
    payload_version INTEGER NOT NULL,
    attempts INTEGER NOT NULL,
    last_error LONGTEXT NOT NULL,
    failed_at DATETIME(6) NOT NULL,
    INDEX idx_outbox_dead_letters_failed_at (failed_at, id),
    INDEX idx_outbox_dead_letters_tenant (tenant_id, failed_at, id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
