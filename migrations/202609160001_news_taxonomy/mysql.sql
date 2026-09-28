CREATE TABLE news_taxonomy_terms (
    id VARCHAR(128) PRIMARY KEY,
    slug VARCHAR(256) NOT NULL,
    kind VARCHAR(16) NOT NULL CHECK (kind IN ('category','tag')),
    parent_id VARCHAR(128),
    labels_json LONGTEXT NOT NULL,
    UNIQUE (slug, kind)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE news_taxonomy_relations (
    content_id VARCHAR(128) NOT NULL,
    term_id VARCHAR(128) NOT NULL,
    PRIMARY KEY (content_id, term_id),
    CONSTRAINT news_taxonomy_content_fk FOREIGN KEY (content_id) REFERENCES news_items(id) ON DELETE CASCADE,
    CONSTRAINT news_taxonomy_term_fk FOREIGN KEY (term_id) REFERENCES news_taxonomy_terms(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
CREATE INDEX idx_news_taxonomy_relation_term ON news_taxonomy_relations(term_id, content_id);
