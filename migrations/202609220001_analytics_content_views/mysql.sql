CREATE TABLE analytics_content_daily (
    day CHAR(10) NOT NULL,
    tenant_id VARCHAR(128) NOT NULL,
    content_type VARCHAR(32) NOT NULL,
    content_id VARCHAR(128) NOT NULL,
    locale VARCHAR(16) NOT NULL DEFAULT '',
    purpose_version VARCHAR(64) NOT NULL,
    view_count BIGINT NOT NULL CHECK (view_count >= 0),
    PRIMARY KEY (day,tenant_id,content_type,content_id,locale,purpose_version)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
