ALTER TABLE form_inbox ADD COLUMN lead_status VARCHAR(16) NOT NULL DEFAULT 'new' CHECK (lead_status IN ('new','contacted','qualified','won','lost'));
ALTER TABLE form_inbox ADD COLUMN assigned_to VARCHAR(128) NOT NULL DEFAULT '';
DROP INDEX uq_form_inbox_idempotency ON form_inbox;
CREATE UNIQUE INDEX uq_form_inbox_tenant_idempotency ON form_inbox(tenant_id,form_key,idempotency_identity);

CREATE TABLE form_lead_notes (
    id VARCHAR(128) PRIMARY KEY,
    tenant_id VARCHAR(128) NOT NULL DEFAULT '',
    submission_id VARCHAR(128) NOT NULL,
    author_id VARCHAR(128) NOT NULL,
    note TEXT NOT NULL CHECK (CHAR_LENGTH(note) BETWEEN 1 AND 8192),
    created_at DATETIME(6) NOT NULL,
    INDEX idx_form_lead_notes_tenant_submission (tenant_id,submission_id,created_at,id),
    CONSTRAINT form_lead_notes_submission_fk FOREIGN KEY (submission_id) REFERENCES form_inbox(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE mail_spool (
    id VARCHAR(128) PRIMARY KEY,
    tenant_id VARCHAR(128) NOT NULL DEFAULT '',
    event_key VARCHAR(128) NOT NULL,
    recipient VARCHAR(320) NOT NULL CHECK (CHAR_LENGTH(recipient) BETWEEN 3 AND 320),
    subject VARCHAR(998) NOT NULL CHECK (CHAR_LENGTH(subject) BETWEEN 1 AND 998),
    body LONGTEXT NOT NULL CHECK (CHAR_LENGTH(body) <= 1048576),
    html_body LONGTEXT NOT NULL,
    status VARCHAR(24) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending','sending','delivered','failed_permanent')),
    attempts INTEGER NOT NULL DEFAULT 0 CHECK (attempts >= 0),
    next_attempt_at DATETIME(6) NOT NULL,
    lease_owner VARCHAR(128),
    lease_token VARCHAR(128),
    lease_until DATETIME(6),
    last_error VARCHAR(2048) NOT NULL DEFAULT '',
    created_at DATETIME(6) NOT NULL,
    updated_at DATETIME(6) NOT NULL,
    delivered_at DATETIME(6),
    UNIQUE (tenant_id,event_key),
    INDEX idx_mail_spool_due (tenant_id,status,next_attempt_at,lease_until,id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TRIGGER form_lead_notes_tenant_guard_insert BEFORE INSERT ON form_lead_notes FOR EACH ROW
BEGIN
    IF NOT EXISTS (SELECT 1 FROM form_inbox WHERE id = NEW.submission_id AND tenant_id = NEW.tenant_id) THEN
        SIGNAL SQLSTATE '45000' SET MESSAGE_TEXT = 'form lead note tenant mismatch';
    END IF;
END;
