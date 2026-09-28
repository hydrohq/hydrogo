CREATE TABLE analytics_daily_stats (
    day CHAR(10) NOT NULL,
    tenant_id VARCHAR(128) NOT NULL DEFAULT '',
    metric VARCHAR(32) NOT NULL CHECK (metric IN ('page','unique_visitors','referrer','device','browser','os')),
    dimension VARCHAR(512) NOT NULL DEFAULT '',
    event_count BIGINT NOT NULL CHECK (event_count >= 0),
    PRIMARY KEY (day,tenant_id,metric,dimension)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
CREATE INDEX idx_analytics_daily_stats_report ON analytics_daily_stats(day,tenant_id,metric,event_count DESC);

CREATE TABLE analytics_daily_visitors (
    day CHAR(10) NOT NULL,
    tenant_id VARCHAR(128) NOT NULL DEFAULT '',
    visitor_key CHAR(64) NOT NULL CHECK (CHAR_LENGTH(visitor_key) = 64),
    PRIMARY KEY (day,tenant_id,visitor_key)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
CREATE INDEX idx_analytics_daily_visitors_retention ON analytics_daily_visitors(day);
