CREATE TABLE restore_controls (
    id INTEGER PRIMARY KEY CHECK (id = 1),
    sessions_revoked INTEGER NOT NULL,
    egress_quarantined INTEGER NOT NULL,
    old_writes_blocked INTEGER NOT NULL,
    activated_at TEXT NOT NULL
);
