CREATE TABLE sectoral_portfolio_tags (
    tenant_id TEXT NOT NULL DEFAULT '',
    content_id TEXT NOT NULL REFERENCES sectoral_portfolio(id) ON DELETE CASCADE,
    tag TEXT NOT NULL CHECK (length(tag) BETWEEN 1 AND 192),
    PRIMARY KEY (tenant_id, content_id, tag)
);
CREATE INDEX sectoral_portfolio_tags_lookup_idx ON sectoral_portfolio_tags(tenant_id, tag, content_id);
ALTER TABLE sectoral_portfolio_tags ENABLE ROW LEVEL SECURITY;
ALTER TABLE sectoral_portfolio_tags FORCE ROW LEVEL SECURITY;
CREATE POLICY hydrogo_tenant_isolation ON sectoral_portfolio_tags
USING ((coalesce(nullif(current_setting('app.tenant_rls_enforced', true), ''), 'false') <> 'true') OR tenant_id = nullif(current_setting('app.current_tenant_id', true), ''))
WITH CHECK ((coalesce(nullif(current_setting('app.tenant_rls_enforced', true), ''), 'false') <> 'true') OR tenant_id = nullif(current_setting('app.current_tenant_id', true), ''));
INSERT INTO sectoral_portfolio_tags (tenant_id, content_id, tag)
SELECT p.tenant_id, p.id, 'tag:' || lower(trim(tags.value))
FROM sectoral_portfolio p
CROSS JOIN LATERAL jsonb_array_elements_text(p.payload::jsonb->'tags') AS tags(value)
WHERE length(trim(tags.value)) BETWEEN 1 AND 64
ON CONFLICT DO NOTHING;
INSERT INTO sectoral_portfolio_tags (tenant_id, content_id, tag)
SELECT tenant_id, id, 'category:' || lower(trim(payload::jsonb->>'industry'))
FROM sectoral_portfolio
WHERE length(trim(payload::jsonb->>'industry')) BETWEEN 1 AND 128
ON CONFLICT DO NOTHING;
