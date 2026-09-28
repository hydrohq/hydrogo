CREATE INDEX idx_news_public_list ON news_items (status, published_revision, locale, updated_at DESC, id);
CREATE INDEX idx_news_admin_list ON news_items (updated_at DESC, id);
