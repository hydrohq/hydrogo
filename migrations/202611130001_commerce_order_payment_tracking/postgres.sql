ALTER TABLE commerce_customer_orders ADD COLUMN IF NOT EXISTS payment_method TEXT NOT NULL DEFAULT '';
ALTER TABLE commerce_customer_orders ADD COLUMN IF NOT EXISTS payment_ref TEXT NOT NULL DEFAULT '';
ALTER TABLE commerce_customer_orders ADD COLUMN IF NOT EXISTS hold_until TIMESTAMPTZ;
