CREATE TABLE sectoral_portfolio_tags (
    tenant_id TEXT NOT NULL DEFAULT '',
    content_id TEXT NOT NULL REFERENCES sectoral_portfolio(id) ON DELETE CASCADE,
    tag TEXT NOT NULL CHECK (length(tag) BETWEEN 1 AND 192),
    PRIMARY KEY (tenant_id, content_id, tag)
);
CREATE INDEX sectoral_portfolio_tags_lookup_idx ON sectoral_portfolio_tags(tenant_id, tag, content_id);
INSERT OR IGNORE INTO sectoral_portfolio_tags (tenant_id, content_id, tag)
SELECT p.tenant_id, p.id, 'tag:' || lower(trim(tags.value))
FROM sectoral_portfolio p, json_each(p.payload, '$.tags') AS tags
WHERE length(trim(tags.value)) BETWEEN 1 AND 64;
INSERT OR IGNORE INTO sectoral_portfolio_tags (tenant_id, content_id, tag)
SELECT tenant_id, id, 'category:' || lower(trim(json_extract(payload, '$.industry')))
FROM sectoral_portfolio
WHERE length(trim(json_extract(payload, '$.industry'))) BETWEEN 1 AND 128;
