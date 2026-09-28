ALTER TABLE page_revisions ADD COLUMN actor_id VARCHAR(128) NOT NULL DEFAULT '';
CREATE INDEX idx_page_revisions_actor ON page_revisions(tenant_id,actor_id,created_at);
