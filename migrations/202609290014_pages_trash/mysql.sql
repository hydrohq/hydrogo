ALTER TABLE pages ADD COLUMN deleted_at DATETIME(6) NULL;
CREATE INDEX idx_pages_deleted_at ON pages(deleted_at);
