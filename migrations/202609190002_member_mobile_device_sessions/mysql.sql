CREATE TABLE member_mobile_device_sessions (
    id VARCHAR(128) PRIMARY KEY,
    member_id VARCHAR(128) NOT NULL,
    device_id VARCHAR(128) NOT NULL,
    device_name VARCHAR(256) NOT NULL DEFAULT '',
    platform VARCHAR(32) NOT NULL DEFAULT '',
    family_id VARCHAR(128) NOT NULL,
    refresh_token_hash CHAR(64) NOT NULL UNIQUE,
    access_token_hash CHAR(64) NOT NULL,
    revoked_at DATETIME(6),
    access_expires_at DATETIME(6) NOT NULL,
    refresh_expires_at DATETIME(6) NOT NULL,
    created_at DATETIME(6) NOT NULL,
    last_used_at DATETIME(6) NOT NULL,
    INDEX idx_member_mobile_sessions_member (member_id),
    INDEX idx_member_mobile_sessions_family (family_id),
    INDEX idx_member_mobile_sessions_access_hash (access_token_hash),
    CONSTRAINT member_mobile_sessions_member_fk FOREIGN KEY (member_id) REFERENCES members(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
