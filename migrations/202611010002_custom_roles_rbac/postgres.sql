CREATE TABLE IF NOT EXISTS hydro_roles (
    tenant_id TEXT NOT NULL DEFAULT '',
    id TEXT NOT NULL,
    name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,
    PRIMARY KEY (tenant_id, id)
);

CREATE TABLE IF NOT EXISTS hydro_role_permissions (
    tenant_id TEXT NOT NULL DEFAULT '',
    role_id TEXT NOT NULL,
    resource TEXT NOT NULL,
    action TEXT NOT NULL,
    PRIMARY KEY (tenant_id, role_id, resource, action),
    FOREIGN KEY (tenant_id, role_id) REFERENCES hydro_roles(tenant_id, id) ON DELETE CASCADE
);

ALTER TABLE hydro_roles ENABLE ROW LEVEL SECURITY;
ALTER TABLE hydro_roles FORCE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS hydrogo_tenant_isolation ON hydro_roles;
CREATE POLICY hydrogo_tenant_isolation ON hydro_roles
    USING ((coalesce(nullif(current_setting('app.tenant_rls_enforced', true), ''), 'false') <> 'true') OR tenant_id = nullif(current_setting('app.current_tenant_id', true), ''))
    WITH CHECK ((coalesce(nullif(current_setting('app.tenant_rls_enforced', true), ''), 'false') <> 'true') OR tenant_id = nullif(current_setting('app.current_tenant_id', true), ''));

ALTER TABLE hydro_role_permissions ENABLE ROW LEVEL SECURITY;
ALTER TABLE hydro_role_permissions FORCE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS hydrogo_tenant_isolation ON hydro_role_permissions;
CREATE POLICY hydrogo_tenant_isolation ON hydro_role_permissions
    USING ((coalesce(nullif(current_setting('app.tenant_rls_enforced', true), ''), 'false') <> 'true') OR tenant_id = nullif(current_setting('app.current_tenant_id', true), ''))
    WITH CHECK ((coalesce(nullif(current_setting('app.tenant_rls_enforced', true), ''), 'false') <> 'true') OR tenant_id = nullif(current_setting('app.current_tenant_id', true), ''));
