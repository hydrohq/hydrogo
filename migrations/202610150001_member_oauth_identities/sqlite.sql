CREATE TABLE member_oauth_identities (
    provider TEXT NOT NULL CHECK (provider IN ('google','github','apple')),
    subject TEXT NOT NULL,
    member_id TEXT NOT NULL REFERENCES members(id) ON DELETE CASCADE,
    tenant_id TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL,
    PRIMARY KEY (provider, subject, tenant_id)
);

CREATE INDEX idx_member_oauth_member ON member_oauth_identities(tenant_id, member_id);
