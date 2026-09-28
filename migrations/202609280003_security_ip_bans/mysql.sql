CREATE TABLE security_ip_bans (
    address VARCHAR(64) PRIMARY KEY,
    expires_at DATETIME(6),
    created_at DATETIME(6) NOT NULL,
    INDEX idx_security_ip_bans_expires (expires_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
