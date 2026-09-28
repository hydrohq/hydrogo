CREATE TABLE analytics_consent (
    consent_id VARCHAR(128) PRIMARY KEY,
    decision VARCHAR(16) NOT NULL CHECK (decision IN ('rejected','granted','revoked')),
    purpose_version VARCHAR(64) NOT NULL,
    decided_at DATETIME(6) NOT NULL,
    revoked_at DATETIME(6)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE analytics_daily (
    day CHAR(10) NOT NULL,
    event_type VARCHAR(64) NOT NULL,
    country CHAR(2) NOT NULL DEFAULT '',
    purpose_version VARCHAR(64) NOT NULL,
    event_count BIGINT NOT NULL CHECK (event_count >= 0),
    duration_seconds BIGINT NOT NULL CHECK (duration_seconds >= 0),
    PRIMARY KEY (day,event_type,country,purpose_version)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
