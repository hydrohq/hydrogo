ALTER TABLE media_assets ADD COLUMN blur_hash TEXT NULL;
ALTER TABLE media_assets ADD COLUMN focal_x DOUBLE NOT NULL DEFAULT 0.5;
ALTER TABLE media_assets ADD COLUMN focal_y DOUBLE NOT NULL DEFAULT 0.5;
ALTER TABLE media_assets ADD COLUMN dominant_colors TEXT NULL;
ALTER TABLE media_assets ADD COLUMN folder_id VARCHAR(128) NULL;
ALTER TABLE media_assets ADD COLUMN tags_json TEXT NULL;
CREATE TABLE media_folders (
  id VARCHAR(128) PRIMARY KEY, tenant_id VARCHAR(128) NOT NULL DEFAULT '', name VARCHAR(256) NOT NULL,
  parent_id VARCHAR(128) NULL, created_at DATETIME(6) NOT NULL, updated_at DATETIME(6) NOT NULL,
  CONSTRAINT media_folders_parent_fk FOREIGN KEY (parent_id) REFERENCES media_folders(id) ON DELETE RESTRICT,
  UNIQUE KEY media_folders_name_parent (tenant_id, parent_id, name),
  INDEX media_folders_tree (tenant_id, parent_id, name)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
CREATE INDEX media_assets_folder ON media_assets(tenant_id, folder_id, updated_at);
