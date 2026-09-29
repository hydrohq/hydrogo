CREATE TABLE member_email_verification_tokens (
    id TEXT PRIMARY KEY,
    member_id TEXT NOT NULL REFERENCES members(id) ON DELETE CASCADE,
    token_hash TEXT NOT NULL UNIQUE,
    expires_at TIMESTAMPTZ NOT NULL,
    consumed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL,
    tenant_id TEXT NOT NULL DEFAULT ''
);
CREATE INDEX idx_member_email_verification_lookup
    ON member_email_verification_tokens(tenant_id, token_hash, expires_at);
ALTER TABLE member_email_verification_tokens ENABLE ROW LEVEL SECURITY;
ALTER TABLE member_email_verification_tokens FORCE ROW LEVEL SECURITY;
CREATE POLICY hydrogo_tenant_isolation ON member_email_verification_tokens
    USING ((coalesce(nullif(current_setting('app.tenant_rls_enforced', true), ''), 'false') <> 'true') OR tenant_id = nullif(current_setting('app.current_tenant_id', true), ''))
    WITH CHECK ((coalesce(nullif(current_setting('app.tenant_rls_enforced', true), ''), 'false') <> 'true') OR tenant_id = nullif(current_setting('app.current_tenant_id', true), ''));
