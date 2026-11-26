-- Adds human-readable short order_number (e.g. 261126-4821) alongside unguessable order id.
ALTER TABLE commerce_customer_orders
    ADD COLUMN order_number VARCHAR(32) NOT NULL DEFAULT '',
    ADD INDEX idx_commerce_customer_orders_number (tenant_id, order_number);
