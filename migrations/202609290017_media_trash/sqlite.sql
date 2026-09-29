ALTER TABLE media_assets ADD COLUMN deleted_at TEXT;
CREATE INDEX idx_media_assets_deleted_at ON media_assets(deleted_at);
