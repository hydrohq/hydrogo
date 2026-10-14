CREATE TABLE member_oauth_identities (
    provider VARCHAR(16) NOT NULL CHECK (provider IN ('google','github','apple')),
    subject VARCHAR(255) NOT NULL,
    member_id VARCHAR(128) NOT NULL,
    tenant_id VARCHAR(128) NOT NULL DEFAULT '',
    created_at DATETIME(6) NOT NULL,
    PRIMARY KEY (provider,subject,tenant_id),
    KEY idx_member_oauth_member (tenant_id,member_id),
    CONSTRAINT fk_member_oauth_member FOREIGN KEY (member_id) REFERENCES members(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
