CREATE TABLE auth_recovery_tokens (
    id VARCHAR(128) PRIMARY KEY,
    user_id VARCHAR(128) NOT NULL,
    purpose VARCHAR(32) NOT NULL,
    token_hash CHAR(64) NOT NULL UNIQUE,
    expires_at DATETIME(6) NOT NULL,
    consumed_at DATETIME(6),
    created_at DATETIME(6) NOT NULL,
    INDEX idx_auth_recovery_lookup (purpose, token_hash),
    CONSTRAINT auth_recovery_user_fk FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
