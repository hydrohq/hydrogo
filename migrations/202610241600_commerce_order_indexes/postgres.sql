CREATE INDEX IF NOT EXISTS idx_commerce_customer_orders_created ON commerce_customer_orders(tenant_id, created_at);
CREATE INDEX IF NOT EXISTS idx_commerce_customer_orders_payment_created ON commerce_customer_orders(tenant_id, payment_status, created_at);
