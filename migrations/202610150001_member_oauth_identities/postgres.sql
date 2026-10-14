CREATE TABLE member_oauth_identities (
    provider TEXT NOT NULL CHECK (provider IN ('google','github','apple')),
    subject TEXT NOT NULL,
    member_id TEXT NOT NULL REFERENCES members(id) ON DELETE CASCADE,
    tenant_id TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (provider, subject, tenant_id)
);

CREATE INDEX idx_member_oauth_member ON member_oauth_identities(tenant_id, member_id);
ALTER TABLE member_oauth_identities ENABLE ROW LEVEL SECURITY;
ALTER TABLE member_oauth_identities FORCE ROW LEVEL SECURITY;
CREATE POLICY hydrogo_tenant_isolation ON member_oauth_identities
    USING ((coalesce(nullif(current_setting('app.tenant_rls_enforced', true), ''), 'false') <> 'true') OR tenant_id = nullif(current_setting('app.current_tenant_id', true), ''))
    WITH CHECK ((coalesce(nullif(current_setting('app.tenant_rls_enforced', true), ''), 'false') <> 'true') OR tenant_id = nullif(current_setting('app.current_tenant_id', true), ''));
