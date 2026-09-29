ALTER TABLE news_items ADD COLUMN deleted_at TIMESTAMPTZ;
CREATE INDEX idx_news_items_deleted_at ON news_items(deleted_at);
