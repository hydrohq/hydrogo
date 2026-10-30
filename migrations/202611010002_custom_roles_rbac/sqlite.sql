CREATE TABLE hydro_roles (
    tenant_id TEXT NOT NULL DEFAULT '',
    id TEXT NOT NULL,
    name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    PRIMARY KEY (tenant_id, id)
);

CREATE TABLE hydro_role_permissions (
    tenant_id TEXT NOT NULL DEFAULT '',
    role_id TEXT NOT NULL,
    resource TEXT NOT NULL,
    action TEXT NOT NULL,
    PRIMARY KEY (tenant_id, role_id, resource, action),
    FOREIGN KEY (tenant_id, role_id) REFERENCES hydro_roles(tenant_id, id) ON DELETE CASCADE
);
