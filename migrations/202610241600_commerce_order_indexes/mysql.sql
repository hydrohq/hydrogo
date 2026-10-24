-- One statement so a partial apply cannot leave one index behind (MySQL
-- auto-commits each DDL statement and has no CREATE INDEX IF NOT EXISTS).
ALTER TABLE commerce_customer_orders
    ADD INDEX idx_commerce_customer_orders_created (tenant_id, created_at),
    ADD INDEX idx_commerce_customer_orders_payment_created (tenant_id, payment_status, created_at);
