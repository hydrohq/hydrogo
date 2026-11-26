-- Adds human-readable short order_number (e.g. 261126-4821) alongside unguessable order id.
ALTER TABLE commerce_customer_orders ADD COLUMN IF NOT EXISTS order_number TEXT NOT NULL DEFAULT '';
CREATE INDEX IF NOT EXISTS idx_commerce_customer_orders_number ON commerce_customer_orders(tenant_id, order_number);
