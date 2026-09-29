CREATE INDEX idx_analytics_view_totals ON analytics_content_daily(tenant_id,content_type,content_id,day,view_count);
