CREATE TABLE analytics_search_daily (
    day DATE NOT NULL,
    tenant_id TEXT NOT NULL DEFAULT '',
    term TEXT NOT NULL,
    locale TEXT NOT NULL DEFAULT 'tr',
    search_count BIGINT NOT NULL DEFAULT 0 CHECK (search_count >= 0),
    zero_result_count BIGINT NOT NULL DEFAULT 0 CHECK (zero_result_count >= 0),
    PRIMARY KEY (day, tenant_id, term, locale)
);
CREATE INDEX idx_analytics_search_report ON analytics_search_daily(day, tenant_id, search_count DESC);
ALTER TABLE analytics_search_daily ENABLE ROW LEVEL SECURITY;
ALTER TABLE analytics_search_daily FORCE ROW LEVEL SECURITY;
CREATE POLICY hydrogo_tenant_isolation ON analytics_search_daily
    USING ((coalesce(nullif(current_setting('app.tenant_rls_enforced', true), ''), 'false') <> 'true') OR tenant_id = nullif(current_setting('app.current_tenant_id', true), ''))
    WITH CHECK ((coalesce(nullif(current_setting('app.tenant_rls_enforced', true), ''), 'false') <> 'true') OR tenant_id = nullif(current_setting('app.current_tenant_id', true), ''));
