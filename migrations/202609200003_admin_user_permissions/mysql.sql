CREATE TABLE admin_user_permissions (
    user_id VARCHAR(128) NOT NULL,
    permission VARCHAR(128) NOT NULL,
    allowed TINYINT(1) NOT NULL,
    tenant_id VARCHAR(128) NOT NULL DEFAULT '',
    created_at DATETIME(6) NOT NULL,
    updated_at DATETIME(6) NOT NULL,
    PRIMARY KEY (user_id, permission, tenant_id),
    INDEX idx_admin_user_permissions_tenant_user (tenant_id, user_id),
    CONSTRAINT admin_user_permissions_user_fk FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
