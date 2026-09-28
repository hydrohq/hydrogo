CREATE TABLE auth_oauth_identities (
    provider TEXT NOT NULL CHECK (provider IN ('google','github','apple')),
    subject TEXT NOT NULL,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    tenant_id TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL,
    PRIMARY KEY (provider, subject, tenant_id)
);

CREATE INDEX idx_auth_oauth_user ON auth_oauth_identities(tenant_id, user_id);
