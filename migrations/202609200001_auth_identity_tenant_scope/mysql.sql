ALTER TABLE users ADD COLUMN tenant_id VARCHAR(128) NOT NULL DEFAULT '';
ALTER TABLE sessions ADD COLUMN tenant_id VARCHAR(128) NOT NULL DEFAULT '';
CREATE INDEX idx_users_tenant_email ON users(tenant_id, email);
CREATE INDEX idx_sessions_tenant_token ON sessions(tenant_id, token_hash);
CREATE INDEX idx_sessions_tenant_user ON sessions(tenant_id, user_id);
