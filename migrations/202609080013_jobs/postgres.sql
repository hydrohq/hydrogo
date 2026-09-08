CREATE TABLE IF NOT EXISTS jobs (
    id TEXT PRIMARY KEY,
    type TEXT NOT NULL,
    payload TEXT NOT NULL,
    available_at TEXT NOT NULL,
    created_at TEXT NOT NULL,
    attempts INTEGER NOT NULL DEFAULT 0,
    lease_owner TEXT,
    lease_token TEXT,
    lease_until TEXT
);
CREATE INDEX IF NOT EXISTS idx_jobs_claim ON jobs (available_at, lease_until, id);
