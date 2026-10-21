-- The audit log belongs to the core: projects without the pages module (the
-- crm preset) get the same table the pages migrations build, in its final
-- shape. Where the pages migrations already created it, every statement here
-- is a no-op.
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

CREATE TRIGGER IF NOT EXISTS audit_events_immutable_update
BEFORE UPDATE ON audit_events
BEGIN
    SELECT RAISE(ABORT, 'audit events are immutable');
END;

CREATE TRIGGER IF NOT EXISTS audit_events_immutable_delete
BEFORE DELETE ON audit_events
BEGIN
    SELECT RAISE(ABORT, 'audit events are immutable');
END;
