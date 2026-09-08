CREATE TABLE news_items (
    id TEXT PRIMARY KEY,
    locale TEXT NOT NULL,
    slug TEXT NOT NULL,
    title TEXT NOT NULL,
    body TEXT NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('draft','scheduled','published','retracted')),
    version INTEGER NOT NULL DEFAULT 1,
    revision INTEGER NOT NULL DEFAULT 1,
    published_revision INTEGER,
    schedule_generation INTEGER NOT NULL DEFAULT 0,
    scheduled_at TEXT,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    UNIQUE(locale, slug)
);
CREATE INDEX idx_news_schedule ON news_items(status, scheduled_at);
CREATE INDEX idx_news_published_route ON news_items(locale, slug, published_revision);
