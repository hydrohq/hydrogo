ALTER TABLE commerce_orders ADD COLUMN idempotency_hash CHAR(64) NOT NULL DEFAULT '';
