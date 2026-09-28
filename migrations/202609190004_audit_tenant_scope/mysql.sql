ALTER TABLE audit_events ADD COLUMN tenant_id VARCHAR(128) NOT NULL DEFAULT '';
CREATE INDEX idx_audit_tenant_target ON audit_events(tenant_id, target_id, id);
