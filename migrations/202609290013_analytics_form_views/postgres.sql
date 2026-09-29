CREATE TABLE analytics_form_daily (
    day DATE NOT NULL,
    tenant_id TEXT NOT NULL DEFAULT '',
    form_id TEXT NOT NULL,
    view_count BIGINT NOT NULL DEFAULT 0 CHECK (view_count >= 0),
    submission_count BIGINT NOT NULL DEFAULT 0 CHECK (submission_count >= 0),
    PRIMARY KEY (day, tenant_id, form_id)
);
CREATE INDEX idx_analytics_form_report ON analytics_form_daily(tenant_id, day, form_id);
ALTER TABLE analytics_form_daily ENABLE ROW LEVEL SECURITY;
ALTER TABLE analytics_form_daily FORCE ROW LEVEL SECURITY;
CREATE POLICY hydrogo_tenant_isolation ON analytics_form_daily
    USING ((coalesce(nullif(current_setting('app.tenant_rls_enforced', true), ''), 'false') <> 'true') OR tenant_id = nullif(current_setting('app.current_tenant_id', true), ''))
    WITH CHECK ((coalesce(nullif(current_setting('app.tenant_rls_enforced', true), ''), 'false') <> 'true') OR tenant_id = nullif(current_setting('app.current_tenant_id', true), ''));
CREATE TABLE analytics_form_visitors (
    day DATE NOT NULL,
    tenant_id TEXT NOT NULL DEFAULT '',
    form_id TEXT NOT NULL,
    visitor_key TEXT NOT NULL,
    PRIMARY KEY (day, tenant_id, form_id, visitor_key)
);
CREATE INDEX idx_analytics_form_visitors_retention ON analytics_form_visitors(day);
ALTER TABLE analytics_form_visitors ENABLE ROW LEVEL SECURITY;
ALTER TABLE analytics_form_visitors FORCE ROW LEVEL SECURITY;
CREATE POLICY hydrogo_tenant_isolation ON analytics_form_visitors
    USING ((coalesce(nullif(current_setting('app.tenant_rls_enforced', true), ''), 'false') <> 'true') OR tenant_id = nullif(current_setting('app.current_tenant_id', true), ''))
    WITH CHECK ((coalesce(nullif(current_setting('app.tenant_rls_enforced', true), ''), 'false') <> 'true') OR tenant_id = nullif(current_setting('app.current_tenant_id', true), ''));
