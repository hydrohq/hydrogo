ALTER TABLE form_inbox ADD COLUMN lead_status TEXT NOT NULL DEFAULT 'new' CHECK (lead_status IN ('new','contacted','qualified','won','lost'));
ALTER TABLE form_inbox ADD COLUMN assigned_to TEXT NOT NULL DEFAULT '';
DROP INDEX IF EXISTS uq_form_inbox_idempotency;
CREATE UNIQUE INDEX uq_form_inbox_tenant_idempotency ON form_inbox(tenant_id,form_key,idempotency_key) WHERE idempotency_key <> '';

CREATE TABLE form_lead_notes (
    id TEXT PRIMARY KEY,
    tenant_id TEXT NOT NULL DEFAULT '',
    submission_id TEXT NOT NULL REFERENCES form_inbox(id) ON DELETE CASCADE,
    author_id TEXT NOT NULL,
    note TEXT NOT NULL CHECK (length(note) BETWEEN 1 AND 8192),
    created_at TIMESTAMPTZ NOT NULL
);
CREATE INDEX idx_form_lead_notes_tenant_submission ON form_lead_notes(tenant_id,submission_id,created_at,id);

CREATE TABLE mail_spool (
    id TEXT PRIMARY KEY,
    tenant_id TEXT NOT NULL DEFAULT '',
    event_key TEXT NOT NULL,
    recipient TEXT NOT NULL CHECK (length(recipient) BETWEEN 3 AND 320),
    subject TEXT NOT NULL CHECK (length(subject) BETWEEN 1 AND 998),
    body TEXT NOT NULL CHECK (length(body) <= 1048576),
    html_body TEXT NOT NULL DEFAULT '' CHECK (length(html_body) <= 1048576),
    status TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending','sending','delivered','failed_permanent')),
    attempts INTEGER NOT NULL DEFAULT 0 CHECK (attempts >= 0),
    next_attempt_at TIMESTAMPTZ NOT NULL,
    lease_owner TEXT,
    lease_token TEXT,
    lease_until TIMESTAMPTZ,
    last_error TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,
    delivered_at TIMESTAMPTZ,
    UNIQUE (tenant_id,event_key)
);
CREATE INDEX idx_mail_spool_due ON mail_spool(tenant_id,status,next_attempt_at,lease_until,id);

ALTER TABLE form_lead_notes ENABLE ROW LEVEL SECURITY;
ALTER TABLE form_lead_notes FORCE ROW LEVEL SECURITY;
CREATE POLICY hydrogo_tenant_isolation ON form_lead_notes
    USING ((coalesce(nullif(current_setting('app.tenant_rls_enforced', true), ''), 'false') <> 'true') OR tenant_id = nullif(current_setting('app.current_tenant_id', true), ''))
    WITH CHECK ((coalesce(nullif(current_setting('app.tenant_rls_enforced', true), ''), 'false') <> 'true') OR tenant_id = nullif(current_setting('app.current_tenant_id', true), ''));
ALTER TABLE mail_spool ENABLE ROW LEVEL SECURITY;
ALTER TABLE mail_spool FORCE ROW LEVEL SECURITY;
CREATE POLICY hydrogo_tenant_isolation ON mail_spool
    USING ((coalesce(nullif(current_setting('app.tenant_rls_enforced', true), ''), 'false') <> 'true') OR tenant_id = nullif(current_setting('app.current_tenant_id', true), ''))
    WITH CHECK ((coalesce(nullif(current_setting('app.tenant_rls_enforced', true), ''), 'false') <> 'true') OR tenant_id = nullif(current_setting('app.current_tenant_id', true), ''));
