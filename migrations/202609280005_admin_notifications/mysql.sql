CREATE TABLE admin_notifications (
    id VARCHAR(128) PRIMARY KEY,
    tenant_id VARCHAR(128),
    user_id VARCHAR(128) NOT NULL,
    title VARCHAR(512) NOT NULL,
    message TEXT NOT NULL,
    target_path VARCHAR(2048) NOT NULL DEFAULT '/admin',
    created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    read_at DATETIME(6),
    dismissed_at DATETIME(6),
    INDEX admin_notifications_inbox (tenant_id,user_id,dismissed_at,created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
