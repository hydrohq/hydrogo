CREATE INDEX idx_pages_public_list ON pages (published_revision, published_locale, updated_at DESC, id);
CREATE INDEX idx_pages_admin_list ON pages (updated_at DESC, id);
