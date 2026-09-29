ALTER TABLE form_inbox ADD COLUMN deleted_at TEXT;
CREATE INDEX idx_form_inbox_deleted_at ON form_inbox(deleted_at);
