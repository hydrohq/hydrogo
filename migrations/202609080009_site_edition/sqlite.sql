CREATE TABLE site_settings (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL,
    is_public INTEGER NOT NULL DEFAULT 0,
    updated_at TEXT NOT NULL
);

CREATE TABLE navigation_items (
    id TEXT PRIMARY KEY,
    locale TEXT NOT NULL,
    label TEXT NOT NULL,
    href TEXT NOT NULL,
    position INTEGER NOT NULL,
    is_visible INTEGER NOT NULL DEFAULT 1,
    updated_at TEXT NOT NULL,
    UNIQUE(locale, position, id)
);
CREATE INDEX idx_navigation_items_locale ON navigation_items(locale, position);

CREATE TABLE page_redirects (
    source_path TEXT PRIMARY KEY,
    target_path TEXT NOT NULL,
    status INTEGER NOT NULL CHECK (status IN (301, 308)),
    created_at TEXT NOT NULL
);

CREATE TABLE form_inbox (
    id TEXT PRIMARY KEY,
    form_key TEXT NOT NULL,
    email TEXT NOT NULL,
    payload TEXT NOT NULL,
    source_ip_hash TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'new',
    created_at TEXT NOT NULL
);
CREATE INDEX idx_form_inbox_key_created ON form_inbox(form_key, created_at);
