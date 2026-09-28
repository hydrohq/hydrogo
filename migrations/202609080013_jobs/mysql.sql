CREATE TABLE IF NOT EXISTS jobs (
    id VARCHAR(128) PRIMARY KEY,
    type VARCHAR(128) NOT NULL,
    payload LONGTEXT NOT NULL,
    available_at DATETIME(6) NOT NULL,
    created_at DATETIME(6) NOT NULL,
    attempts INTEGER NOT NULL DEFAULT 0,
    lease_owner VARCHAR(128),
    lease_token VARCHAR(128),
    lease_until DATETIME(6),
    INDEX idx_jobs_claim (available_at, lease_until, id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
