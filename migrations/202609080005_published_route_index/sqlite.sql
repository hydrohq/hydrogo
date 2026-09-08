DROP INDEX IF EXISTS idx_pages_locale_slug;
CREATE UNIQUE INDEX idx_pages_locale_slug ON pages(locale, slug) WHERE slug <> '' AND status = 'published';
