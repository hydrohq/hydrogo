CREATE TABLE page_tags (
    tenant_id TEXT NOT NULL DEFAULT '',
    page_id TEXT NOT NULL REFERENCES pages(id) ON DELETE CASCADE,
    tag TEXT NOT NULL CHECK (length(tag) BETWEEN 1 AND 64),
    PRIMARY KEY (tenant_id, page_id, tag)
);
CREATE INDEX page_tags_lookup_idx ON page_tags(tenant_id, tag, page_id);
CREATE TABLE page_published_tags (
    tenant_id TEXT NOT NULL DEFAULT '',
    page_id TEXT NOT NULL REFERENCES pages(id) ON DELETE CASCADE,
    tag TEXT NOT NULL CHECK (length(tag) BETWEEN 1 AND 64),
    PRIMARY KEY (tenant_id, page_id, tag)
);
CREATE INDEX page_published_tags_lookup_idx ON page_published_tags(tenant_id, tag, page_id);
ALTER TABLE page_tags ENABLE ROW LEVEL SECURITY;
ALTER TABLE page_tags FORCE ROW LEVEL SECURITY;
CREATE POLICY hydrogo_tenant_isolation ON page_tags
USING ((coalesce(nullif(current_setting('app.tenant_rls_enforced', true), ''), 'false') <> 'true') OR tenant_id = nullif(current_setting('app.current_tenant_id', true), ''))
WITH CHECK ((coalesce(nullif(current_setting('app.tenant_rls_enforced', true), ''), 'false') <> 'true') OR tenant_id = nullif(current_setting('app.current_tenant_id', true), ''));
ALTER TABLE page_published_tags ENABLE ROW LEVEL SECURITY;
ALTER TABLE page_published_tags FORCE ROW LEVEL SECURITY;
CREATE POLICY hydrogo_tenant_isolation ON page_published_tags
USING ((coalesce(nullif(current_setting('app.tenant_rls_enforced', true), ''), 'false') <> 'true') OR tenant_id = nullif(current_setting('app.current_tenant_id', true), ''))
WITH CHECK ((coalesce(nullif(current_setting('app.tenant_rls_enforced', true), ''), 'false') <> 'true') OR tenant_id = nullif(current_setting('app.current_tenant_id', true), ''));
