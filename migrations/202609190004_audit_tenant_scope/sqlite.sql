ALTER TABLE audit_events ADD COLUMN tenant_id TEXT NOT NULL DEFAULT '';
CREATE INDEX idx_audit_tenant_target ON audit_events(tenant_id, target_id, id);
