ALTER TABLE pages ADD COLUMN deleted_at TIMESTAMPTZ;
CREATE INDEX idx_pages_deleted_at ON pages(deleted_at);
