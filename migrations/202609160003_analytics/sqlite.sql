CREATE TABLE analytics_consent (
  consent_id TEXT PRIMARY KEY,
  decision TEXT NOT NULL CHECK (decision IN ('rejected','granted','revoked')),
  purpose_version TEXT NOT NULL,
  decided_at TEXT NOT NULL,
  revoked_at TEXT NULL
);
CREATE TABLE analytics_daily (
  day TEXT NOT NULL,
  event_type TEXT NOT NULL,
  country TEXT NOT NULL DEFAULT '',
  purpose_version TEXT NOT NULL,
  event_count INTEGER NOT NULL CHECK (event_count >= 0),
  duration_seconds INTEGER NOT NULL CHECK (duration_seconds >= 0),
  PRIMARY KEY (day,event_type,country,purpose_version)
);
