CREATE TABLE page_tags (
    tenant_id VARCHAR(128) NOT NULL DEFAULT '',
    page_id VARCHAR(128) NOT NULL,
    tag VARCHAR(64) NOT NULL CHECK (CHAR_LENGTH(tag) BETWEEN 1 AND 64),
    PRIMARY KEY (tenant_id,page_id,tag),
    KEY page_tags_lookup_idx (tenant_id,tag,page_id),
    CONSTRAINT fk_page_tags_page FOREIGN KEY (page_id) REFERENCES pages(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
CREATE TABLE page_published_tags (
    tenant_id VARCHAR(128) NOT NULL DEFAULT '',
    page_id VARCHAR(128) NOT NULL,
    tag VARCHAR(64) NOT NULL CHECK (CHAR_LENGTH(tag) BETWEEN 1 AND 64),
    PRIMARY KEY (tenant_id,page_id,tag),
    KEY page_published_tags_lookup_idx (tenant_id,tag,page_id),
    CONSTRAINT fk_page_published_tags_page FOREIGN KEY (page_id) REFERENCES pages(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
