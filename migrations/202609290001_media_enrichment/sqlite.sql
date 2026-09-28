ALTER TABLE media_assets ADD COLUMN blur_hash TEXT NOT NULL DEFAULT '';
ALTER TABLE media_assets ADD COLUMN focal_x REAL NOT NULL DEFAULT 0.5;
ALTER TABLE media_assets ADD COLUMN focal_y REAL NOT NULL DEFAULT 0.5;
ALTER TABLE media_assets ADD COLUMN dominant_colors TEXT NOT NULL DEFAULT '[]';
ALTER TABLE media_assets ADD COLUMN folder_id TEXT;
ALTER TABLE media_assets ADD COLUMN tags_json TEXT NOT NULL DEFAULT '[]';
CREATE TABLE media_folders (
  id TEXT PRIMARY KEY, tenant_id TEXT NOT NULL DEFAULT '', name TEXT NOT NULL,
  parent_id TEXT REFERENCES media_folders(id) ON DELETE RESTRICT,
  created_at TEXT NOT NULL, updated_at TEXT NOT NULL
);
CREATE UNIQUE INDEX media_folders_name_parent ON media_folders(tenant_id, parent_id, name);
CREATE INDEX media_folders_tree ON media_folders(tenant_id, parent_id, name);
CREATE INDEX media_assets_folder ON media_assets(tenant_id, folder_id, updated_at);
