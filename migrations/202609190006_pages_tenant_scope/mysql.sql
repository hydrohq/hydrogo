ALTER TABLE pages ADD COLUMN tenant_id VARCHAR(128) NOT NULL DEFAULT '';
ALTER TABLE page_revisions ADD COLUMN tenant_id VARCHAR(128) NOT NULL DEFAULT '';
ALTER TABLE pages DROP INDEX idx_pages_locale_slug;
CREATE UNIQUE INDEX idx_pages_tenant_locale_slug ON pages(tenant_id, locale, published_slug_identity);
CREATE INDEX idx_pages_tenant_public_list ON pages(tenant_id, published_revision, published_locale, updated_at DESC, id);
CREATE INDEX idx_page_revisions_tenant_page ON page_revisions(tenant_id, page_id, revision);
