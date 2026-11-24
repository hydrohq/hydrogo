UPDATE admin_notifications SET tenant_id = '' WHERE tenant_id IS NULL;
ALTER TABLE admin_notifications ALTER COLUMN tenant_id SET DEFAULT '';
ALTER TABLE admin_notifications ALTER COLUMN tenant_id SET NOT NULL;
ALTER TABLE admin_notifications ENABLE ROW LEVEL SECURITY;
ALTER TABLE admin_notifications FORCE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS hydrogo_tenant_isolation ON admin_notifications;
CREATE POLICY hydrogo_tenant_isolation ON admin_notifications
    FOR ALL
    USING (tenant_id = coalesce(current_setting('app.current_tenant_id', true), ''))
    WITH CHECK (tenant_id = coalesce(current_setting('app.current_tenant_id', true), ''));
ALTER TABLE admin_digest_preferences ENABLE ROW LEVEL SECURITY;
ALTER TABLE admin_digest_preferences FORCE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS hydrogo_tenant_isolation ON admin_digest_preferences;
CREATE POLICY hydrogo_tenant_isolation ON admin_digest_preferences
    FOR ALL
    USING (tenant_id = coalesce(current_setting('app.current_tenant_id', true), ''))
    WITH CHECK (tenant_id = coalesce(current_setting('app.current_tenant_id', true), ''));
