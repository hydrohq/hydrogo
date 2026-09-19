ALTER TABLE members ADD COLUMN tenant_id TEXT NOT NULL DEFAULT '';
ALTER TABLE member_sessions ADD COLUMN tenant_id TEXT NOT NULL DEFAULT '';
ALTER TABLE member_mobile_device_sessions ADD COLUMN tenant_id TEXT NOT NULL DEFAULT '';
CREATE INDEX idx_members_tenant_email ON members(tenant_id, email);
CREATE INDEX idx_member_sessions_tenant_member ON member_sessions(tenant_id, member_id);
CREATE INDEX idx_member_mobile_sessions_tenant_member ON member_mobile_device_sessions(tenant_id, member_id);
