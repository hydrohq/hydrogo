CREATE TABLE analytics_form_daily (
    day TEXT NOT NULL,
    tenant_id TEXT NOT NULL DEFAULT '',
    form_id TEXT NOT NULL,
    view_count INTEGER NOT NULL DEFAULT 0 CHECK (view_count >= 0),
    submission_count INTEGER NOT NULL DEFAULT 0 CHECK (submission_count >= 0),
    PRIMARY KEY (day, tenant_id, form_id)
);
CREATE INDEX idx_analytics_form_report ON analytics_form_daily(tenant_id, day, form_id);
CREATE TABLE analytics_form_visitors (
    day TEXT NOT NULL,
    tenant_id TEXT NOT NULL DEFAULT '',
    form_id TEXT NOT NULL,
    visitor_key TEXT NOT NULL,
    PRIMARY KEY (day, tenant_id, form_id, visitor_key)
);
CREATE INDEX idx_analytics_form_visitors_retention ON analytics_form_visitors(day);
