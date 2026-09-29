CREATE TABLE analytics_feedback_daily (
    tenant_id VARCHAR(128) NOT NULL,
    content_type VARCHAR(16) NOT NULL,
    content_id VARCHAR(128) NOT NULL,
    day CHAR(10) NOT NULL,
    reaction VARCHAR(4) NOT NULL,
    event_count BIGINT NOT NULL DEFAULT 0,
    PRIMARY KEY (tenant_id, content_type, content_id, day, reaction),
    CONSTRAINT chk_analytics_feedback_reaction CHECK (reaction IN ('up','down')),
    CONSTRAINT chk_analytics_feedback_count CHECK (event_count >= 0)
);
CREATE INDEX idx_analytics_feedback_day ON analytics_feedback_daily(day);
