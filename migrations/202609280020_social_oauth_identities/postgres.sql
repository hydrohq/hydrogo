CREATE TABLE auth_oauth_identities (
    provider TEXT NOT NULL CHECK (provider IN ('google','github','apple')),
    subject TEXT NOT NULL,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    tenant_id TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (provider, subject, tenant_id)
);

CREATE INDEX idx_auth_oauth_user ON auth_oauth_identities(tenant_id, user_id);
ALTER TABLE auth_oauth_identities ENABLE ROW LEVEL SECURITY;
ALTER TABLE auth_oauth_identities FORCE ROW LEVEL SECURITY;
CREATE POLICY auth_oauth_tenant_policy ON auth_oauth_identities
    USING (current_setting('app.tenant_rls_enforced', true) = 'on'
       AND tenant_id = current_setting('app.current_tenant_id', true))
    WITH CHECK (current_setting('app.tenant_rls_enforced', true) = 'on'
       AND tenant_id = current_setting('app.current_tenant_id', true));
