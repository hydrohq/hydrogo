ALTER TABLE form_inbox ADD COLUMN idempotency_key VARCHAR(128) NOT NULL DEFAULT '';
ALTER TABLE form_inbox ADD COLUMN request_hash CHAR(64) NOT NULL DEFAULT '';
ALTER TABLE form_inbox ADD COLUMN idempotency_identity VARCHAR(128)
    GENERATED ALWAYS AS (NULLIF(idempotency_key, '')) STORED;
CREATE UNIQUE INDEX uq_form_inbox_idempotency ON form_inbox(form_key, idempotency_identity);
