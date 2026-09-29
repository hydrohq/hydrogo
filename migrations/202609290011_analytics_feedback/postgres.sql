CREATE TABLE analytics_feedback_daily (
    tenant_id TEXT NOT NULL,
    content_type TEXT NOT NULL,
    content_id TEXT NOT NULL,
    day TEXT NOT NULL,
    reaction TEXT NOT NULL CHECK (reaction IN ('up','down')),
    event_count BIGINT NOT NULL DEFAULT 0 CHECK (event_count >= 0),
    PRIMARY KEY (tenant_id, content_type, content_id, day, reaction)
);
CREATE INDEX idx_analytics_feedback_day ON analytics_feedback_daily(day);
