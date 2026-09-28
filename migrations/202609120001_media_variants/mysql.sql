CREATE TABLE media_variants (
    id VARCHAR(128) PRIMARY KEY,
    media_id VARCHAR(128) NOT NULL,
    kind VARCHAR(32) NOT NULL,
    storage_key VARCHAR(512) NOT NULL UNIQUE,
    mime_type VARCHAR(128) NOT NULL,
    size_bytes BIGINT NOT NULL,
    width INTEGER NOT NULL DEFAULT 0,
    height INTEGER NOT NULL DEFAULT 0,
    duration_ms BIGINT NOT NULL DEFAULT 0,
    status VARCHAR(16) NOT NULL DEFAULT 'processing',
    error_message VARCHAR(1024) NOT NULL DEFAULT '',
    created_at DATETIME(6) NOT NULL,
    updated_at DATETIME(6) NOT NULL,
    UNIQUE (media_id, kind),
    INDEX idx_media_variants_media_id (media_id),
    CONSTRAINT media_variants_media_fk FOREIGN KEY (media_id) REFERENCES media_assets(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
