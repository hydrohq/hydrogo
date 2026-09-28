ALTER TABLE auth_invites ADD COLUMN tenant_id VARCHAR(128) NOT NULL DEFAULT '';
ALTER TABLE auth_recovery_tokens ADD COLUMN tenant_id VARCHAR(128) NOT NULL DEFAULT '';
ALTER TABLE user_mfa ADD COLUMN tenant_id VARCHAR(128) NOT NULL DEFAULT '';
ALTER TABLE mfa_recovery_codes ADD COLUMN tenant_id VARCHAR(128) NOT NULL DEFAULT '';
CREATE INDEX idx_auth_invites_tenant_token ON auth_invites(tenant_id, token_hash);
CREATE INDEX idx_auth_recovery_tenant_token ON auth_recovery_tokens(tenant_id, purpose, token_hash);
CREATE INDEX idx_user_mfa_tenant_user ON user_mfa(tenant_id, user_id);
CREATE INDEX idx_mfa_codes_tenant_user ON mfa_recovery_codes(tenant_id, user_id);
