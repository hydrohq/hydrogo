CREATE INDEX idx_pages_public_list ON pages(published_locale, updated_at DESC, id) WHERE published_revision IS NOT NULL;
CREATE INDEX idx_pages_admin_list ON pages(updated_at DESC, id);
