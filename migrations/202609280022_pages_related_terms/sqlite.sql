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
