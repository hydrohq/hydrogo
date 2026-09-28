CREATE TABLE auth_invites (
    id VARCHAR(128) PRIMARY KEY,
    email VARCHAR(320) NOT NULL,
    role VARCHAR(16) NOT NULL CHECK (role IN ('editor', 'viewer')),
    token_hash CHAR(64) NOT NULL UNIQUE,
    expires_at DATETIME(6) NOT NULL,
    consumed_at DATETIME(6),
    created_at DATETIME(6) NOT NULL,
    INDEX idx_auth_invites_lookup (token_hash)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
