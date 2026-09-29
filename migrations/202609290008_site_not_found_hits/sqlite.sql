CREATE TABLE site_not_found_hits (
    tenant_id TEXT NOT NULL DEFAULT '',
    path_hash TEXT NOT NULL CHECK (length(path_hash) = 64),
    path TEXT NOT NULL CHECK (length(path) <= 2048),
    hit_count INTEGER NOT NULL DEFAULT 1 CHECK (hit_count > 0),
    first_seen TEXT NOT NULL,
    last_seen TEXT NOT NULL,
    PRIMARY KEY (tenant_id,path_hash)
);
CREATE INDEX idx_site_not_found_hits_rank ON site_not_found_hits(tenant_id,hit_count DESC,last_seen DESC);
