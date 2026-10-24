-- The admin order list and the member order history sort newest first, and
-- the expiry sweeper scans pending orders by age; without created_at in an
-- index each of those sorts the whole table.
CREATE INDEX IF NOT EXISTS idx_commerce_customer_orders_created ON commerce_customer_orders(tenant_id, created_at);
CREATE INDEX IF NOT EXISTS idx_commerce_customer_orders_payment_created ON commerce_customer_orders(tenant_id, payment_status, created_at);
