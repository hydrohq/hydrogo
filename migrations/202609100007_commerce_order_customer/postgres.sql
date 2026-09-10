ALTER TABLE commerce_orders ADD COLUMN customer_name TEXT NOT NULL DEFAULT '';
ALTER TABLE commerce_orders ADD COLUMN customer_identity_number TEXT NOT NULL DEFAULT '';
ALTER TABLE commerce_orders ADD COLUMN customer_email TEXT NOT NULL DEFAULT '';
ALTER TABLE commerce_orders ADD COLUMN customer_phone TEXT NOT NULL DEFAULT '';
ALTER TABLE commerce_orders ADD COLUMN customer_address TEXT NOT NULL DEFAULT '';
ALTER TABLE commerce_orders ADD COLUMN customer_city TEXT NOT NULL DEFAULT '';
ALTER TABLE commerce_orders ADD COLUMN customer_country TEXT NOT NULL DEFAULT '';
ALTER TABLE commerce_orders ADD COLUMN customer_postal_code TEXT NOT NULL DEFAULT '';
