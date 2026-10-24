ALTER TABLE commerce_customer_orders ADD COLUMN IF NOT EXISTS shipping_carrier TEXT NOT NULL DEFAULT '';
ALTER TABLE commerce_customer_orders ADD COLUMN IF NOT EXISTS shipping_tracking_number TEXT NOT NULL DEFAULT '';
ALTER TABLE commerce_customer_orders ADD COLUMN IF NOT EXISTS shipping_tracking_url TEXT NOT NULL DEFAULT '';
ALTER TABLE commerce_customer_orders ADD COLUMN IF NOT EXISTS shipped_at TIMESTAMPTZ;
