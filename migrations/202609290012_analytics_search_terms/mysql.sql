CREATE TABLE analytics_search_daily (
    day DATE NOT NULL,
    tenant_id VARCHAR(128) NOT NULL DEFAULT '',
    term VARCHAR(128) NOT NULL,
    locale VARCHAR(35) NOT NULL DEFAULT 'tr',
    search_count BIGINT NOT NULL DEFAULT 0,
    zero_result_count BIGINT NOT NULL DEFAULT 0,
    PRIMARY KEY (day, tenant_id, term, locale),
    INDEX idx_analytics_search_report (tenant_id, day, search_count DESC),
    CONSTRAINT chk_analytics_search_counts CHECK (search_count >= 0 AND zero_result_count >= 0)
);
