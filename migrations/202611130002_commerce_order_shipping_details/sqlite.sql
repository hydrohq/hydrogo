-- Stores provider shipment id, printable label URL, and delivery timestamp on customer orders.
ALTER TABLE commerce_customer_orders ADD COLUMN shipping_shipment_id TEXT NOT NULL DEFAULT '';
ALTER TABLE commerce_customer_orders ADD COLUMN shipping_label_url TEXT NOT NULL DEFAULT '';
ALTER TABLE commerce_customer_orders ADD COLUMN delivered_at TEXT;
CREATE INDEX IF NOT EXISTS idx_commerce_customer_orders_shipment ON commerce_customer_orders(tenant_id, shipping_shipment_id);
