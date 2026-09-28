ALTER TABLE outbox_events ADD COLUMN available_at DATETIME(6) NOT NULL DEFAULT '1970-01-01 00:00:00.000000';
ALTER TABLE outbox_events ADD COLUMN attempts INTEGER NOT NULL DEFAULT 0;
ALTER TABLE outbox_events ADD COLUMN lease_owner VARCHAR(128);
ALTER TABLE outbox_events ADD COLUMN lease_token VARCHAR(128);
ALTER TABLE outbox_events ADD COLUMN lease_until DATETIME(6);
ALTER TABLE outbox_events ADD COLUMN last_error LONGTEXT;
CREATE INDEX idx_outbox_claimable ON outbox_events(available_at, lease_until, id);
