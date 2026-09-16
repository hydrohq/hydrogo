ALTER TABLE form_inbox ADD COLUMN idempotency_key TEXT NOT NULL DEFAULT '';
ALTER TABLE form_inbox ADD COLUMN request_hash TEXT NOT NULL DEFAULT '';
CREATE UNIQUE INDEX uq_form_inbox_idempotency ON form_inbox(form_key, idempotency_key) WHERE idempotency_key <> '';
