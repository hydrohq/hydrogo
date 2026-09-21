CREATE TABLE tenants (
    tenant_id TEXT PRIMARY KEY,
    state TEXT NOT NULL CHECK (state IN ('provisioning', 'active')),
    default_locale TEXT NOT NULL,
    timezone TEXT NOT NULL,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);
CREATE TABLE tenant_system_audit (
    id TEXT PRIMARY KEY,
    tenant_id TEXT NOT NULL,
    action TEXT NOT NULL,
    created_at TEXT NOT NULL
);
CREATE INDEX idx_tenant_system_audit_tenant_created ON tenant_system_audit(tenant_id, created_at);
