ALTER TABLE media_assets ADD COLUMN tenant_id TEXT NOT NULL DEFAULT '';
ALTER TABLE media_variants ADD COLUMN tenant_id TEXT NOT NULL DEFAULT '';
CREATE INDEX idx_media_assets_tenant_updated ON media_assets(tenant_id, updated_at, id);
CREATE INDEX idx_media_variants_tenant_media ON media_variants(tenant_id, media_id, kind);
