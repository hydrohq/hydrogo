CREATE TABLE auth_oauth_identities (
    provider VARCHAR(16) NOT NULL CHECK (provider IN ('google','github','apple')),
    subject VARCHAR(255) NOT NULL,
    user_id VARCHAR(128) NOT NULL,
    tenant_id VARCHAR(128) NOT NULL DEFAULT '',
    created_at DATETIME(6) NOT NULL,
    PRIMARY KEY (provider,subject,tenant_id),
    KEY idx_auth_oauth_user (tenant_id,user_id),
    CONSTRAINT fk_auth_oauth_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
