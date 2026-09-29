CREATE TABLE analytics_form_daily (
    day DATE NOT NULL,
    tenant_id VARCHAR(128) NOT NULL DEFAULT '',
    form_id VARCHAR(128) NOT NULL,
    view_count BIGINT NOT NULL DEFAULT 0,
    submission_count BIGINT NOT NULL DEFAULT 0,
    PRIMARY KEY (day, tenant_id, form_id),
    INDEX idx_analytics_form_report (tenant_id, day, form_id),
    CONSTRAINT chk_analytics_form_counts CHECK (view_count >= 0 AND submission_count >= 0)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
CREATE TABLE analytics_form_visitors (
    day DATE NOT NULL,
    tenant_id VARCHAR(128) NOT NULL DEFAULT '',
    form_id VARCHAR(128) NOT NULL,
    visitor_key VARCHAR(64) NOT NULL,
    PRIMARY KEY (day, tenant_id, form_id, visitor_key),
    INDEX idx_analytics_form_visitors_retention (day)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
