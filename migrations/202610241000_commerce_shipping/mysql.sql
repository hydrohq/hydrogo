ALTER TABLE commerce_customer_orders ADD COLUMN shipping_carrier VARCHAR(32) NOT NULL DEFAULT '';
ALTER TABLE commerce_customer_orders ADD COLUMN shipping_tracking_number VARCHAR(128) NOT NULL DEFAULT '';
ALTER TABLE commerce_customer_orders ADD COLUMN shipping_tracking_url VARCHAR(512) NOT NULL DEFAULT '';
ALTER TABLE commerce_customer_orders ADD COLUMN shipped_at DATETIME(6);
