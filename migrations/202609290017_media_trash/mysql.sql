ALTER TABLE media_assets ADD COLUMN deleted_at DATETIME(6) NULL;
CREATE INDEX idx_media_assets_deleted_at ON media_assets(deleted_at);
