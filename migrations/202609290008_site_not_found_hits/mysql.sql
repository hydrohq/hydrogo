CREATE TABLE site_not_found_hits (
    tenant_id VARCHAR(128) NOT NULL DEFAULT '',
    path_hash CHAR(64) NOT NULL,
    path VARCHAR(2048) NOT NULL,
    hit_count BIGINT NOT NULL DEFAULT 1,
    first_seen DATETIME(6) NOT NULL,
    last_seen DATETIME(6) NOT NULL,
    PRIMARY KEY (tenant_id,path_hash),
    INDEX idx_site_not_found_hits_rank (tenant_id,hit_count,last_seen)
);
