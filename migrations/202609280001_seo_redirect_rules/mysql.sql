CREATE TABLE seo_redirect_rules (
    tenant_id VARCHAR(128) NOT NULL DEFAULT '',
    id VARCHAR(128) NOT NULL,
    pattern VARCHAR(512) NOT NULL,
    match_kind VARCHAR(16) NOT NULL CHECK (match_kind IN ('exact','wildcard','regex')),
    target_path VARCHAR(2048) NOT NULL,
    status INTEGER NOT NULL CHECK (status IN (301,302,308)),
    hits BIGINT NOT NULL DEFAULT 0 CHECK (hits >= 0),
    created_at DATETIME(6) NOT NULL,
    PRIMARY KEY (tenant_id,id),
    UNIQUE (tenant_id,match_kind,pattern),
    INDEX idx_seo_redirect_rules_tenant_kind (tenant_id,match_kind,pattern)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
