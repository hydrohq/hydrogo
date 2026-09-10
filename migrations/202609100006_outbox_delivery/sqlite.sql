ALTER TABLE outbox_events ADD COLUMN available_at TEXT NOT NULL DEFAULT '1970-01-01T00:00:00Z';
ALTER TABLE outbox_events ADD COLUMN attempts INTEGER NOT NULL DEFAULT 0;
ALTER TABLE outbox_events ADD COLUMN lease_owner TEXT;
ALTER TABLE outbox_events ADD COLUMN lease_token TEXT;
ALTER TABLE outbox_events ADD COLUMN lease_until TEXT;
ALTER TABLE outbox_events ADD COLUMN last_error TEXT;
CREATE INDEX idx_outbox_claimable ON outbox_events(available_at, lease_until, id);
