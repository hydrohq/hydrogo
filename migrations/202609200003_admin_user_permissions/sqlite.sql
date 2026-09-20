CREATE TABLE admin_user_permissions (
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    permission TEXT NOT NULL,
    allowed INTEGER NOT NULL,
    tenant_id TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    PRIMARY KEY (user_id, permission, tenant_id)
);

CREATE INDEX idx_admin_user_permissions_tenant_user ON admin_user_permissions(tenant_id, user_id);
