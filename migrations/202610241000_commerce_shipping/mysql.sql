-- One statement: MySQL commits each DDL statement on its own, so four
-- separate ALTERs that failed halfway would leave columns behind with no
-- ledger row, and the rerun would die on "Duplicate column name".
ALTER TABLE commerce_customer_orders
    ADD COLUMN shipping_carrier VARCHAR(32) NOT NULL DEFAULT '',
    ADD COLUMN shipping_tracking_number VARCHAR(128) NOT NULL DEFAULT '',
    ADD COLUMN shipping_tracking_url VARCHAR(512) NOT NULL DEFAULT '',
    ADD COLUMN shipped_at DATETIME(6);
