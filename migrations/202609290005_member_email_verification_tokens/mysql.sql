CREATE TABLE member_email_verification_tokens (
    id VARCHAR(128) PRIMARY KEY,
    member_id VARCHAR(128) NOT NULL,
    token_hash VARCHAR(64) NOT NULL UNIQUE,
    expires_at DATETIME(6) NOT NULL,
    consumed_at DATETIME(6) NULL,
    created_at DATETIME(6) NOT NULL,
    tenant_id VARCHAR(128) NOT NULL DEFAULT '',
    CONSTRAINT member_email_verification_tokens_member_fk
        FOREIGN KEY (member_id) REFERENCES members(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
CREATE INDEX idx_member_email_verification_lookup
    ON member_email_verification_tokens(tenant_id, token_hash, expires_at);
