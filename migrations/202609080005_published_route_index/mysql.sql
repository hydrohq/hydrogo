ALTER TABLE pages DROP INDEX idx_pages_locale_slug;
ALTER TABLE pages ADD COLUMN published_slug_identity VARCHAR(256)
    GENERATED ALWAYS AS (CASE WHEN slug <> '' AND status = 'published' THEN slug ELSE NULL END) STORED;
CREATE UNIQUE INDEX idx_pages_locale_slug ON pages(locale, published_slug_identity);
