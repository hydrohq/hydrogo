ALTER TABLE pages ADD COLUMN tenant_id TEXT NOT NULL DEFAULT '';
ALTER TABLE page_revisions ADD COLUMN tenant_id TEXT NOT NULL DEFAULT '';
DROP INDEX IF EXISTS idx_pages_locale_slug;
CREATE UNIQUE INDEX idx_pages_tenant_locale_slug ON pages(tenant_id, locale, slug) WHERE slug <> '' AND status = 'published';
CREATE INDEX idx_pages_tenant_public_list ON pages(tenant_id, published_locale, updated_at DESC, id) WHERE published_revision IS NOT NULL;
CREATE INDEX idx_page_revisions_tenant_page ON page_revisions(tenant_id, page_id, revision);
