ALTER TABLE commerce_orders ADD COLUMN idempotency_hash TEXT NOT NULL DEFAULT '';
