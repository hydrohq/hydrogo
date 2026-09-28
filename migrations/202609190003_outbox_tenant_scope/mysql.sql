ALTER TABLE outbox_events ADD COLUMN tenant_id VARCHAR(128) NOT NULL DEFAULT '';
CREATE INDEX idx_outbox_tenant_claimable ON outbox_events(tenant_id, available_at, lease_until, id);
