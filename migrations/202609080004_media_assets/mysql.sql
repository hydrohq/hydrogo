CREATE TABLE media_assets (
    id VARCHAR(128) PRIMARY KEY,
    storage_key VARCHAR(512) NOT NULL UNIQUE,
    original_filename VARCHAR(512) NOT NULL,
    mime_type VARCHAR(128) NOT NULL,
    size_bytes BIGINT NOT NULL,
    width INTEGER NOT NULL DEFAULT 0,
    height INTEGER NOT NULL DEFAULT 0,
    visibility VARCHAR(16) NOT NULL DEFAULT 'private',
    status VARCHAR(16) NOT NULL DEFAULT 'pending',
    created_at DATETIME(6) NOT NULL,
    updated_at DATETIME(6) NOT NULL,
    INDEX idx_media_assets_status (status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
