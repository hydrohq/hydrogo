CREATE TABLE api_tokens (
    id VARCHAR(128) PRIMARY KEY,
    user_id VARCHAR(128) NOT NULL,
    tenant_id VARCHAR(128) NOT NULL DEFAULT '',
    name VARCHAR(256) NOT NULL,
    kind VARCHAR(16) NOT NULL,
    token_hash CHAR(64) NOT NULL UNIQUE,
    token_prefix VARCHAR(16) NOT NULL,
    scopes VARCHAR(256) NOT NULL DEFAULT '',
    expires_at DATETIME(6),
    last_used_at DATETIME(6),
    created_at DATETIME(6) NOT NULL,
    revoked_at DATETIME(6),
    INDEX idx_api_tokens_tenant_user (tenant_id, user_id),
    CONSTRAINT api_tokens_user_fk FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    CONSTRAINT api_tokens_kind_chk CHECK (kind IN ('personal','api_key'))
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
