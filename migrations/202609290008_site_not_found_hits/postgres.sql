CREATE TABLE site_not_found_hits (
    tenant_id TEXT NOT NULL DEFAULT '',
    path_hash CHAR(64) NOT NULL CHECK (length(path_hash) = 64),
    path TEXT NOT NULL CHECK (length(path) <= 2048),
    hit_count BIGINT NOT NULL DEFAULT 1 CHECK (hit_count > 0),
    first_seen TIMESTAMPTZ NOT NULL,
    last_seen TIMESTAMPTZ NOT NULL,
    PRIMARY KEY (tenant_id,path_hash)
);
CREATE INDEX idx_site_not_found_hits_rank ON site_not_found_hits(tenant_id,hit_count DESC,last_seen DESC);
ALTER TABLE site_not_found_hits ENABLE ROW LEVEL SECURITY;
ALTER TABLE site_not_found_hits FORCE ROW LEVEL SECURITY;
CREATE POLICY hydrogo_tenant_isolation ON site_not_found_hits
    USING (tenant_id = current_setting('hydrogo.tenant_id', true))
    WITH CHECK (tenant_id = current_setting('hydrogo.tenant_id', true));
