CREATE TABLE analytics_content_daily (
  day DATE NOT NULL,
  tenant_id TEXT NOT NULL,
  content_type TEXT NOT NULL,
  content_id TEXT NOT NULL,
  locale TEXT NOT NULL DEFAULT '',
  purpose_version TEXT NOT NULL,
  view_count BIGINT NOT NULL CHECK (view_count >= 0),
  PRIMARY KEY (day,tenant_id,content_type,content_id,locale,purpose_version)
);
