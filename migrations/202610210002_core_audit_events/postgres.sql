-- The audit log belongs to the core: projects without the pages module (the
-- crm preset) get the same table the pages migrations build, in its final
-- shape. Where the pages migrations already created it, every statement here
-- leaves the existing table, indexes, triggers and policy as they are.
CREATE TABLE IF NOT EXISTS audit_events (
    id TEXT PRIMARY KEY,
    action TEXT NOT NULL,
    target_id TEXT NOT NULL,
    request_id TEXT,
    tenant_id TEXT NOT NULL DEFAULT '',
    actor_id TEXT NOT NULL DEFAULT '',
    resource_type TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL DEFAULT '1970-01-01T00:00:00.000000000Z',
    before_json TEXT NOT NULL DEFAULT '{}',
    after_json TEXT NOT NULL DEFAULT '{}'
);
CREATE INDEX IF NOT EXISTS idx_audit_tenant_target ON audit_events(tenant_id, target_id, id);
CREATE INDEX IF NOT EXISTS idx_audit_events_tenant_created ON audit_events(tenant_id, created_at, id);

CREATE OR REPLACE FUNCTION hydro_reject_audit_event_mutation() RETURNS trigger AS $$
BEGIN
    RAISE EXCEPTION 'audit events are immutable';
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS audit_events_immutable_update ON audit_events;
CREATE TRIGGER audit_events_immutable_update
BEFORE UPDATE OR DELETE ON audit_events
FOR EACH ROW EXECUTE FUNCTION hydro_reject_audit_event_mutation();

ALTER TABLE audit_events ENABLE ROW LEVEL SECURITY;
ALTER TABLE audit_events FORCE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS hydrogo_tenant_isolation ON audit_events;
CREATE POLICY hydrogo_tenant_isolation ON audit_events
    USING ((coalesce(nullif(current_setting('app.tenant_rls_enforced', true), ''), 'false') <> 'true') OR tenant_id = nullif(current_setting('app.current_tenant_id', true), ''))
    WITH CHECK ((coalesce(nullif(current_setting('app.tenant_rls_enforced', true), ''), 'false') <> 'true') OR tenant_id = nullif(current_setting('app.current_tenant_id', true), ''));
