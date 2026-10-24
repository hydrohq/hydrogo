ALTER TABLE commerce_customer_orders ADD COLUMN IF NOT EXISTS guest_district TEXT NOT NULL DEFAULT '';
