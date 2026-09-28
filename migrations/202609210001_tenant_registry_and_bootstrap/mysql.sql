CREATE TABLE tenants (
    tenant_id VARCHAR(128) PRIMARY KEY,
    state VARCHAR(16) NOT NULL CHECK (state IN ('provisioning', 'active')),
    default_locale VARCHAR(16) NOT NULL,
    timezone VARCHAR(64) NOT NULL,
    created_at DATETIME(6) NOT NULL,
    updated_at DATETIME(6) NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE tenant_system_audit (
    id VARCHAR(128) PRIMARY KEY,
    tenant_id VARCHAR(128) NOT NULL,
    action VARCHAR(128) NOT NULL,
    created_at DATETIME(6) NOT NULL,
    INDEX idx_tenant_system_audit_tenant_created (tenant_id, created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
