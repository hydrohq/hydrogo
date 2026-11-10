CREATE TABLE IF NOT EXISTS notification_outbox (
    id TEXT PRIMARY KEY,
    channel TEXT NOT NULL,
    payload TEXT NOT NULL,
    attempts INTEGER NOT NULL DEFAULT 0,
    next_attempt_at TIMESTAMPTZ NOT NULL,
    sent_at TIMESTAMPTZ,
    last_error TEXT NOT NULL DEFAULT '',
    tenant_id TEXT NOT NULL DEFAULT ''
);
CREATE INDEX IF NOT EXISTS idx_notification_outbox_poll ON notification_outbox(sent_at, next_attempt_at);
CREATE INDEX IF NOT EXISTS idx_notification_outbox_tenant ON notification_outbox(tenant_id, sent_at, next_attempt_at);

ALTER TABLE notification_outbox ENABLE ROW LEVEL SECURITY;
ALTER TABLE notification_outbox FORCE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS hydrogo_tenant_isolation ON notification_outbox;
CREATE POLICY hydrogo_tenant_isolation ON notification_outbox
    USING ((coalesce(nullif(current_setting('app.tenant_rls_enforced', true), ''), 'false') <> 'true') OR tenant_id = nullif(current_setting('app.current_tenant_id', true), ''))
    WITH CHECK ((coalesce(nullif(current_setting('app.tenant_rls_enforced', true), ''), 'false') <> 'true') OR tenant_id = nullif(current_setting('app.current_tenant_id', true), ''));
