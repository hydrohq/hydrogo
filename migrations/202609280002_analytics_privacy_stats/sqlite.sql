CREATE TABLE analytics_daily_stats (
    day TEXT NOT NULL,
    tenant_id TEXT NOT NULL DEFAULT '',
    metric TEXT NOT NULL CHECK (metric IN ('page','unique_visitors','referrer','device','browser','os')),
    dimension TEXT NOT NULL DEFAULT '',
    event_count INTEGER NOT NULL CHECK (event_count >= 0),
    PRIMARY KEY (day,tenant_id,metric,dimension)
);
CREATE INDEX idx_analytics_daily_stats_report ON analytics_daily_stats(day,tenant_id,metric,event_count DESC);
CREATE TABLE analytics_daily_visitors (
    day TEXT NOT NULL,
    tenant_id TEXT NOT NULL DEFAULT '',
    visitor_key TEXT NOT NULL CHECK (length(visitor_key) = 64),
    PRIMARY KEY (day,tenant_id,visitor_key)
);
CREATE INDEX idx_analytics_daily_visitors_retention ON analytics_daily_visitors(day);
