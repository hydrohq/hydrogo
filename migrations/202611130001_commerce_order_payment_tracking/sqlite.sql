-- Tracks the chosen payment method, the latest provider attempt reference
-- (such as PayTR's per-attempt merchant_oid on retry), and an optional extended
-- reservation hold timestamp (such as while a card payment is under fraud review).
ALTER TABLE commerce_customer_orders ADD COLUMN payment_method TEXT NOT NULL DEFAULT '';
ALTER TABLE commerce_customer_orders ADD COLUMN payment_ref TEXT NOT NULL DEFAULT '';
ALTER TABLE commerce_customer_orders ADD COLUMN hold_until TEXT;
