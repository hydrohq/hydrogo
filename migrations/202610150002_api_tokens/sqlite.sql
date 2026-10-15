CREATE TABLE api_tokens (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    tenant_id TEXT NOT NULL DEFAULT '',
    name TEXT NOT NULL,
    kind TEXT NOT NULL CHECK (kind IN ('personal','api_key')),
    token_hash TEXT NOT NULL UNIQUE,
    token_prefix TEXT NOT NULL,
    scopes TEXT NOT NULL DEFAULT '',
    expires_at TEXT,
    last_used_at TEXT,
    created_at TEXT NOT NULL,
    revoked_at TEXT
);

CREATE INDEX idx_api_tokens_tenant_user ON api_tokens(tenant_id, user_id);
