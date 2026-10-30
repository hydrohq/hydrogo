CREATE TABLE IF NOT EXISTS hydro_roles (
    tenant_id VARCHAR(128) NOT NULL DEFAULT '',
    id VARCHAR(64) NOT NULL,
    name VARCHAR(128) NOT NULL,
    description TEXT,
    created_at DATETIME(6) NOT NULL,
    updated_at DATETIME(6) NOT NULL,
    PRIMARY KEY (tenant_id, id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS hydro_role_permissions (
    tenant_id VARCHAR(128) NOT NULL DEFAULT '',
    role_id VARCHAR(64) NOT NULL,
    resource VARCHAR(128) NOT NULL,
    action VARCHAR(64) NOT NULL,
    PRIMARY KEY (tenant_id, role_id, resource, action),
    FOREIGN KEY (tenant_id, role_id) REFERENCES hydro_roles(tenant_id, id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
