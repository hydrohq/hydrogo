ALTER TABLE news_items ADD COLUMN expires_at TIMESTAMPTZ;
CREATE INDEX idx_news_expiry ON news_items(status, expires_at);
