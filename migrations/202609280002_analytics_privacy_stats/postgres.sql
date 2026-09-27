CREATE TABLE analytics_daily_stats (
    day DATE NOT NULL,
    tenant_id TEXT NOT NULL DEFAULT '',
    metric TEXT NOT NULL CHECK (metric IN ('page','unique_visitors','referrer','device','browser','os')),
    dimension TEXT NOT NULL DEFAULT '',
    event_count BIGINT NOT NULL CHECK (event_count >= 0),
    PRIMARY KEY (day,tenant_id,metric,dimension)
);
CREATE INDEX idx_analytics_daily_stats_report ON analytics_daily_stats(day,tenant_id,metric,event_count DESC);
ALTER TABLE analytics_daily_stats ENABLE ROW LEVEL SECURITY;
ALTER TABLE analytics_daily_stats FORCE ROW LEVEL SECURITY;
CREATE POLICY hydrogo_tenant_isolation ON analytics_daily_stats
    USING ((coalesce(nullif(current_setting('app.tenant_rls_enforced', true), ''), 'false') <> 'true') OR tenant_id = nullif(current_setting('app.current_tenant_id', true), ''))
    WITH CHECK ((coalesce(nullif(current_setting('app.tenant_rls_enforced', true), ''), 'false') <> 'true') OR tenant_id = nullif(current_setting('app.current_tenant_id', true), ''));
CREATE TABLE analytics_daily_visitors (
    day DATE NOT NULL,
    tenant_id TEXT NOT NULL DEFAULT '',
    visitor_key CHAR(64) NOT NULL,
    PRIMARY KEY (day,tenant_id,visitor_key)
);
CREATE INDEX idx_analytics_daily_visitors_retention ON analytics_daily_visitors(day);
ALTER TABLE analytics_daily_visitors ENABLE ROW LEVEL SECURITY;
ALTER TABLE analytics_daily_visitors FORCE ROW LEVEL SECURITY;
CREATE POLICY hydrogo_tenant_isolation ON analytics_daily_visitors
    USING ((coalesce(nullif(current_setting('app.tenant_rls_enforced', true), ''), 'false') <> 'true') OR tenant_id = nullif(current_setting('app.current_tenant_id', true), ''))
    WITH CHECK ((coalesce(nullif(current_setting('app.tenant_rls_enforced', true), ''), 'false') <> 'true') OR tenant_id = nullif(current_setting('app.current_tenant_id', true), ''));
