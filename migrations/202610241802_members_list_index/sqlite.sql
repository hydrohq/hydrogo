CREATE INDEX idx_members_tenant_list ON members(tenant_id, created_at DESC, id);
