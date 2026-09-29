CREATE TABLE analytics_search_daily (
    day TEXT NOT NULL,
    tenant_id TEXT NOT NULL DEFAULT '',
    term TEXT NOT NULL,
    locale TEXT NOT NULL DEFAULT 'tr',
    search_count INTEGER NOT NULL DEFAULT 0 CHECK (search_count >= 0),
    zero_result_count INTEGER NOT NULL DEFAULT 0 CHECK (zero_result_count >= 0),
    PRIMARY KEY (day, tenant_id, term, locale)
);
CREATE INDEX idx_analytics_search_report ON analytics_search_daily(day, tenant_id, search_count DESC);
