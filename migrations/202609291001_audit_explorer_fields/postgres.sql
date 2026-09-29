ALTER TABLE audit_events ADD COLUMN actor_id TEXT NOT NULL DEFAULT '';
ALTER TABLE audit_events ADD COLUMN resource_type TEXT NOT NULL DEFAULT '';
ALTER TABLE audit_events ADD COLUMN created_at TEXT NOT NULL DEFAULT '1970-01-01T00:00:00.000000000Z';
ALTER TABLE audit_events ADD COLUMN before_json TEXT NOT NULL DEFAULT '{}';
ALTER TABLE audit_events ADD COLUMN after_json TEXT NOT NULL DEFAULT '{}';
CREATE INDEX idx_audit_events_tenant_created ON audit_events(tenant_id, created_at, id);
