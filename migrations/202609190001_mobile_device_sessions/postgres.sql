CREATE TABLE mobile_device_sessions (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    device_id TEXT NOT NULL,
    device_name TEXT NOT NULL DEFAULT '',
    platform TEXT NOT NULL DEFAULT '',
    family_id TEXT NOT NULL,
    refresh_token_hash TEXT NOT NULL UNIQUE,
    access_token_hash TEXT NOT NULL,
    revoked_at TIMESTAMPTZ,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    last_used_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_mobile_sessions_user ON mobile_device_sessions(user_id);
CREATE INDEX idx_mobile_sessions_family ON mobile_device_sessions(family_id);
CREATE INDEX idx_mobile_sessions_refresh_hash ON mobile_device_sessions(refresh_token_hash);
CREATE INDEX idx_mobile_sessions_access_hash ON mobile_device_sessions(access_token_hash);
