CREATE TABLE member_mobile_device_sessions (
    id TEXT PRIMARY KEY,
    member_id TEXT NOT NULL REFERENCES members(id) ON DELETE CASCADE,
    device_id TEXT NOT NULL,
    device_name TEXT NOT NULL DEFAULT '',
    platform TEXT NOT NULL DEFAULT '',
    family_id TEXT NOT NULL,
    refresh_token_hash TEXT NOT NULL UNIQUE,
    access_token_hash TEXT NOT NULL,
    revoked_at TEXT,
    access_expires_at TEXT NOT NULL,
    refresh_expires_at TEXT NOT NULL,
    created_at TEXT NOT NULL,
    last_used_at TEXT NOT NULL
);

CREATE INDEX idx_member_mobile_sessions_member ON member_mobile_device_sessions(member_id);
CREATE INDEX idx_member_mobile_sessions_family ON member_mobile_device_sessions(family_id);
CREATE INDEX idx_member_mobile_sessions_access_hash ON member_mobile_device_sessions(access_token_hash);
