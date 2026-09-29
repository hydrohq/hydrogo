ALTER TABLE users ADD COLUMN last_login_at TEXT;
ALTER TABLE users ADD COLUMN last_login_ip TEXT NOT NULL DEFAULT '';
ALTER TABLE users ADD COLUMN previous_login_at TEXT;
ALTER TABLE users ADD COLUMN previous_login_ip TEXT NOT NULL DEFAULT '';
ALTER TABLE users ADD COLUMN failed_login_attempts INTEGER NOT NULL DEFAULT 0 CHECK (failed_login_attempts >= 0);
ALTER TABLE users ADD COLUMN failed_attempts_since_last_login INTEGER NOT NULL DEFAULT 0 CHECK (failed_attempts_since_last_login >= 0);
