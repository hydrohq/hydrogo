CREATE TABLE admin_notifications (
    id TEXT PRIMARY KEY,
    tenant_id TEXT,
    user_id TEXT NOT NULL,
    title TEXT NOT NULL,
    message TEXT NOT NULL,
    target_path TEXT NOT NULL DEFAULT '/admin',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    read_at TIMESTAMPTZ,
    dismissed_at TIMESTAMPTZ
);
CREATE INDEX admin_notifications_inbox ON admin_notifications(tenant_id, user_id, dismissed_at, created_at);
