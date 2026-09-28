CREATE TABLE user_mfa (
    user_id VARCHAR(128) PRIMARY KEY,
    secret_ciphertext BLOB NOT NULL,
    enabled TINYINT(1) NOT NULL DEFAULT 0,
    created_at DATETIME(6) NOT NULL,
    updated_at DATETIME(6) NOT NULL,
    CONSTRAINT user_mfa_user_fk FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE mfa_recovery_codes (
    id VARCHAR(128) PRIMARY KEY,
    user_id VARCHAR(128) NOT NULL,
    code_hash CHAR(64) NOT NULL UNIQUE,
    used_at DATETIME(6),
    INDEX idx_mfa_recovery_user (user_id),
    CONSTRAINT mfa_recovery_user_fk FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
