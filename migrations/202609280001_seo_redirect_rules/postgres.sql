CREATE TABLE seo_redirect_rules (
    tenant_id TEXT NOT NULL DEFAULT '',
    id TEXT NOT NULL,
    pattern TEXT NOT NULL,
    match_kind TEXT NOT NULL CHECK (match_kind IN ('exact','wildcard','regex')),
    target_path TEXT NOT NULL,
    status INTEGER NOT NULL CHECK (status IN (301,302,308)),
    hits BIGINT NOT NULL DEFAULT 0 CHECK (hits >= 0),
    created_at TIMESTAMPTZ NOT NULL,
    PRIMARY KEY (tenant_id,id),
    UNIQUE (tenant_id,match_kind,pattern)
);
CREATE INDEX idx_seo_redirect_rules_tenant_kind ON seo_redirect_rules(tenant_id,match_kind,pattern);
ALTER TABLE seo_redirect_rules ENABLE ROW LEVEL SECURITY;
ALTER TABLE seo_redirect_rules FORCE ROW LEVEL SECURITY;
CREATE POLICY hydrogo_tenant_isolation ON seo_redirect_rules
    USING ((coalesce(nullif(current_setting('app.tenant_rls_enforced', true), ''), 'false') <> 'true') OR tenant_id = nullif(current_setting('app.current_tenant_id', true), ''))
    WITH CHECK ((coalesce(nullif(current_setting('app.tenant_rls_enforced', true), ''), 'false') <> 'true') OR tenant_id = nullif(current_setting('app.current_tenant_id', true), ''));
