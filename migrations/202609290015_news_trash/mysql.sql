ALTER TABLE news_items ADD COLUMN deleted_at DATETIME(6) NULL;
CREATE INDEX idx_news_items_deleted_at ON news_items(deleted_at);
