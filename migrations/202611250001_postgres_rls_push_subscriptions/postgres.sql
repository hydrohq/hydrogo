ALTER TABLE push_subscriptions ENABLE ROW LEVEL SECURITY;
ALTER TABLE push_subscriptions FORCE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS hydrogo_tenant_isolation ON push_subscriptions;
CREATE POLICY hydrogo_tenant_isolation ON push_subscriptions
    FOR ALL
    USING (tenant_id = coalesce(current_setting('app.current_tenant_id', true), ''))
    WITH CHECK (tenant_id = coalesce(current_setting('app.current_tenant_id', true), ''));
