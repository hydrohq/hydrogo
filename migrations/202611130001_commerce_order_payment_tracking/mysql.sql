ALTER TABLE commerce_customer_orders
    ADD COLUMN payment_method VARCHAR(64) NOT NULL DEFAULT '',
    ADD COLUMN payment_ref VARCHAR(128) NOT NULL DEFAULT '',
    ADD COLUMN hold_until DATETIME(6);
