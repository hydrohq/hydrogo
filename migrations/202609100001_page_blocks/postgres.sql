ALTER TABLE pages ADD COLUMN blocks JSONB NOT NULL DEFAULT '[]'::jsonb;
ALTER TABLE pages ADD COLUMN published_blocks JSONB NOT NULL DEFAULT '[]'::jsonb;
ALTER TABLE page_revisions ADD COLUMN blocks JSONB NOT NULL DEFAULT '[]'::jsonb;
