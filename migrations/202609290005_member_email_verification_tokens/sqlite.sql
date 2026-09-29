CREATE TABLE member_email_verification_tokens (
    id TEXT PRIMARY KEY,
    member_id TEXT NOT NULL REFERENCES members(id) ON DELETE CASCADE,
    token_hash TEXT NOT NULL UNIQUE,
    expires_at TEXT NOT NULL,
    consumed_at TEXT,
    created_at TEXT NOT NULL,
    tenant_id TEXT NOT NULL DEFAULT ''
);
CREATE INDEX idx_member_email_verification_lookup
    ON member_email_verification_tokens(tenant_id, token_hash, expires_at);
