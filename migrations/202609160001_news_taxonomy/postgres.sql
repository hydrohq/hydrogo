CREATE TABLE news_taxonomy_terms (
  id TEXT PRIMARY KEY,
  slug TEXT NOT NULL,
  kind TEXT NOT NULL CHECK (kind IN ('category','tag')),
  parent_id TEXT NULL,
  labels_json TEXT NOT NULL,
  UNIQUE (slug, kind)
);
CREATE TABLE news_taxonomy_relations (
  content_id TEXT NOT NULL,
  term_id TEXT NOT NULL,
  PRIMARY KEY (content_id, term_id),
  FOREIGN KEY (content_id) REFERENCES news_items(id) ON DELETE CASCADE,
  FOREIGN KEY (term_id) REFERENCES news_taxonomy_terms(id) ON DELETE CASCADE
);
CREATE INDEX idx_news_taxonomy_relation_term ON news_taxonomy_relations(term_id, content_id);
