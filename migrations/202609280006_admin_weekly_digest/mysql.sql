CREATE TABLE admin_digest_preferences (
    tenant_id VARCHAR(128) NOT NULL DEFAULT '',
    user_id VARCHAR(128) NOT NULL,
    enabled TINYINT(1) NOT NULL DEFAULT 0 CHECK (enabled IN (0,1)),
    timezone VARCHAR(64) NOT NULL DEFAULT 'UTC',
    last_sent_week VARCHAR(16) NOT NULL DEFAULT '',
    updated_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    PRIMARY KEY (tenant_id,user_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
