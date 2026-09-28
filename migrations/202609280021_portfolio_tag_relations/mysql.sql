CREATE TABLE sectoral_portfolio_tags (
    tenant_id VARCHAR(128) NOT NULL DEFAULT '',
    content_id VARCHAR(128) NOT NULL,
    tag VARCHAR(192) NOT NULL CHECK (CHAR_LENGTH(tag) BETWEEN 1 AND 192),
    PRIMARY KEY (tenant_id,content_id,tag),
    KEY sectoral_portfolio_tags_lookup_idx (tenant_id,tag,content_id),
    CONSTRAINT fk_sectoral_portfolio_tags_content FOREIGN KEY (content_id) REFERENCES sectoral_portfolio(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
INSERT IGNORE INTO sectoral_portfolio_tags (tenant_id,content_id,tag)
WITH RECURSIVE tag_indexes (n) AS (
    SELECT 0 UNION ALL SELECT n + 1 FROM tag_indexes WHERE n < 63
)
SELECT p.tenant_id,p.id,CONCAT('tag:',LOWER(TRIM(JSON_UNQUOTE(JSON_EXTRACT(p.payload,CONCAT('$.tags[',tag_indexes.n,']'))))))
FROM sectoral_portfolio p
CROSS JOIN tag_indexes
WHERE JSON_TYPE(JSON_EXTRACT(p.payload,CONCAT('$.tags[',tag_indexes.n,']'))) = 'STRING'
  AND CHAR_LENGTH(TRIM(JSON_UNQUOTE(JSON_EXTRACT(p.payload,CONCAT('$.tags[',tag_indexes.n,']'))))) BETWEEN 1 AND 64;
INSERT IGNORE INTO sectoral_portfolio_tags (tenant_id,content_id,tag)
SELECT tenant_id,id,CONCAT('category:',LOWER(TRIM(JSON_UNQUOTE(JSON_EXTRACT(payload,'$.industry')))))
FROM sectoral_portfolio
WHERE CHAR_LENGTH(TRIM(JSON_UNQUOTE(JSON_EXTRACT(payload,'$.industry')))) BETWEEN 1 AND 128;
