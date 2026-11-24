ALTER TABLE outbox_dead_letters ENABLE ROW LEVEL SECURITY;
ALTER TABLE outbox_dead_letters FORCE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS hydrogo_tenant_isolation ON outbox_dead_letters;
CREATE POLICY hydrogo_tenant_isolation ON outbox_dead_letters
    FOR ALL
    USING (tenant_id = coalesce(current_setting('app.current_tenant_id', true), ''))
    WITH CHECK (tenant_id = coalesce(current_setting('app.current_tenant_id', true), ''));
