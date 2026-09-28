CREATE TABLE news_items (
    id VARCHAR(128) PRIMARY KEY,
    locale VARCHAR(16) NOT NULL,
    slug VARCHAR(256) NOT NULL,
    title VARCHAR(512) NOT NULL,
    body LONGTEXT NOT NULL,
    status VARCHAR(16) NOT NULL CHECK (status IN ('draft','scheduled','published','retracted')),
    version INTEGER NOT NULL DEFAULT 1,
    revision INTEGER NOT NULL DEFAULT 1,
    published_revision INTEGER,
    schedule_generation BIGINT NOT NULL DEFAULT 0,
    scheduled_at DATETIME(6),
    created_at DATETIME(6) NOT NULL,
    updated_at DATETIME(6) NOT NULL,
    UNIQUE (locale, slug),
    INDEX idx_news_schedule (status, scheduled_at),
    INDEX idx_news_published_route (locale, slug, published_revision)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
