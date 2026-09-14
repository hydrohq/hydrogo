CREATE INDEX idx_news_public_list ON news_items(locale, updated_at DESC, id) WHERE status = 'published' AND published_revision IS NOT NULL;
CREATE INDEX idx_news_admin_list ON news_items(updated_at DESC, id);
