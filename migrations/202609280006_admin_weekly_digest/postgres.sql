CREATE TABLE admin_digest_preferences (
    tenant_id TEXT NOT NULL DEFAULT '',
    user_id TEXT NOT NULL,
    enabled BOOLEAN NOT NULL DEFAULT FALSE,
    timezone TEXT NOT NULL DEFAULT 'UTC',
    last_sent_week TEXT NOT NULL DEFAULT '',
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (tenant_id,user_id)
);
