ALTER TABLE news_taxonomy_terms ADD COLUMN tenant_id TEXT NOT NULL DEFAULT '';
ALTER TABLE news_taxonomy_relations ADD COLUMN tenant_id TEXT NOT NULL DEFAULT '';
CREATE INDEX idx_news_taxonomy_terms_tenant ON news_taxonomy_terms(tenant_id, kind, slug);
CREATE INDEX idx_news_taxonomy_relations_tenant ON news_taxonomy_relations(tenant_id, content_id);
