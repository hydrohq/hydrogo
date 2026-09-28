ALTER TABLE commerce_orders ADD COLUMN customer_name VARCHAR(256) NOT NULL DEFAULT '';
ALTER TABLE commerce_orders ADD COLUMN customer_identity_number VARCHAR(64) NOT NULL DEFAULT '';
ALTER TABLE commerce_orders ADD COLUMN customer_email VARCHAR(320) NOT NULL DEFAULT '';
ALTER TABLE commerce_orders ADD COLUMN customer_phone VARCHAR(64) NOT NULL DEFAULT '';
ALTER TABLE commerce_orders ADD COLUMN customer_address VARCHAR(512) NOT NULL DEFAULT '';
ALTER TABLE commerce_orders ADD COLUMN customer_city VARCHAR(128) NOT NULL DEFAULT '';
ALTER TABLE commerce_orders ADD COLUMN customer_country VARCHAR(128) NOT NULL DEFAULT '';
ALTER TABLE commerce_orders ADD COLUMN customer_postal_code VARCHAR(32) NOT NULL DEFAULT '';
