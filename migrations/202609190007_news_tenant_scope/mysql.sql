ALTER TABLE news_items ADD COLUMN tenant_id VARCHAR(128) NOT NULL DEFAULT '';
CREATE INDEX idx_news_tenant_schedule ON news_items(tenant_id, status, scheduled_at);
CREATE INDEX idx_news_tenant_published_route ON news_items(tenant_id, locale, slug, published_revision);
CREATE INDEX idx_news_tenant_public_list ON news_items(tenant_id, locale, status, published_revision, updated_at DESC, id);
CREATE INDEX idx_news_tenant_admin_list ON news_items(tenant_id, updated_at DESC, id);
