ALTER TABLE commerce_customer_orders ADD COLUMN IF NOT EXISTS shipping_shipment_id TEXT NOT NULL DEFAULT '';
ALTER TABLE commerce_customer_orders ADD COLUMN IF NOT EXISTS shipping_label_url TEXT NOT NULL DEFAULT '';
ALTER TABLE commerce_customer_orders ADD COLUMN IF NOT EXISTS delivered_at TIMESTAMPTZ;
CREATE INDEX IF NOT EXISTS idx_commerce_customer_orders_shipment ON commerce_customer_orders(tenant_id, shipping_shipment_id);
