ALTER TABLE analytics_content_daily ENABLE ROW LEVEL SECURITY;
ALTER TABLE analytics_content_daily FORCE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS hydrogo_tenant_isolation ON analytics_content_daily;
CREATE POLICY hydrogo_tenant_isolation ON analytics_content_daily
    FOR ALL
    USING (tenant_id = coalesce(current_setting('app.current_tenant_id', true), ''))
    WITH CHECK (tenant_id = coalesce(current_setting('app.current_tenant_id', true), ''));
ALTER TABLE analytics_feedback_daily ENABLE ROW LEVEL SECURITY;
ALTER TABLE analytics_feedback_daily FORCE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS hydrogo_tenant_isolation ON analytics_feedback_daily;
CREATE POLICY hydrogo_tenant_isolation ON analytics_feedback_daily
    FOR ALL
    USING (tenant_id = coalesce(current_setting('app.current_tenant_id', true), ''))
    WITH CHECK (tenant_id = coalesce(current_setting('app.current_tenant_id', true), ''));
