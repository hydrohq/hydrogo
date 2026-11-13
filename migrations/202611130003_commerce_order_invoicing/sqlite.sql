-- Stores invoice status, provider metadata, external document link, and claim/issue timestamps on customer orders.
ALTER TABLE commerce_customer_orders ADD COLUMN invoice_status TEXT NOT NULL DEFAULT 'none';
ALTER TABLE commerce_customer_orders ADD COLUMN invoice_provider TEXT NOT NULL DEFAULT '';
ALTER TABLE commerce_customer_orders ADD COLUMN invoice_guid TEXT NOT NULL DEFAULT '';
ALTER TABLE commerce_customer_orders ADD COLUMN invoice_no TEXT NOT NULL DEFAULT '';
ALTER TABLE commerce_customer_orders ADD COLUMN invoice_url TEXT NOT NULL DEFAULT '';
ALTER TABLE commerce_customer_orders ADD COLUMN invoice_error TEXT NOT NULL DEFAULT '';
ALTER TABLE commerce_customer_orders ADD COLUMN invoice_claimed_at TEXT;
ALTER TABLE commerce_customer_orders ADD COLUMN invoiced_at TEXT;
CREATE INDEX IF NOT EXISTS idx_commerce_customer_orders_invoice_due ON commerce_customer_orders(tenant_id, payment_status, invoice_status);
