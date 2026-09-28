ALTER TABLE news_items ADD COLUMN expires_at DATETIME(6) NULL;
CREATE INDEX idx_news_expiry ON news_items(status, expires_at);
