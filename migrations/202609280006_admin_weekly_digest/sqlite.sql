CREATE TABLE admin_digest_preferences (
    tenant_id TEXT NOT NULL DEFAULT '',
    user_id TEXT NOT NULL,
    enabled INTEGER NOT NULL DEFAULT 0 CHECK (enabled IN (0,1)),
    timezone TEXT NOT NULL DEFAULT 'UTC',
    last_sent_week TEXT NOT NULL DEFAULT '',
    updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')),
    PRIMARY KEY (tenant_id,user_id)
);
