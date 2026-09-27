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
    created_at TEXT NOT NULL
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
    next_attempt_at TEXT NOT NULL,
    lease_owner TEXT,
    lease_token TEXT,
    lease_until TEXT,
    last_error TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    delivered_at TEXT,
    UNIQUE (tenant_id,event_key)
);
CREATE INDEX idx_mail_spool_due ON mail_spool(tenant_id,status,next_attempt_at,lease_until,id);

CREATE TRIGGER form_lead_notes_tenant_guard_insert
BEFORE INSERT ON form_lead_notes
WHEN NEW.tenant_id <> (SELECT tenant_id FROM form_inbox WHERE id = NEW.submission_id)
BEGIN
    SELECT RAISE(ABORT, 'form lead note tenant mismatch');
END;
