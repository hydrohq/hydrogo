CREATE TABLE restore_controls (
    id INTEGER PRIMARY KEY CHECK (id = 1),
    sessions_revoked TINYINT(1) NOT NULL,
    egress_quarantined TINYINT(1) NOT NULL,
    old_writes_blocked TINYINT(1) NOT NULL,
    activated_at DATETIME(6) NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
