ALTER TABLE commerce_customer_orders ADD COLUMN IF NOT EXISTS invoice_status TEXT NOT NULL DEFAULT 'none';
ALTER TABLE commerce_customer_orders ADD COLUMN IF NOT EXISTS invoice_provider TEXT NOT NULL DEFAULT '';
ALTER TABLE commerce_customer_orders ADD COLUMN IF NOT EXISTS invoice_guid TEXT NOT NULL DEFAULT '';
ALTER TABLE commerce_customer_orders ADD COLUMN IF NOT EXISTS invoice_no TEXT NOT NULL DEFAULT '';
ALTER TABLE commerce_customer_orders ADD COLUMN IF NOT EXISTS invoice_url TEXT NOT NULL DEFAULT '';
ALTER TABLE commerce_customer_orders ADD COLUMN IF NOT EXISTS invoice_error TEXT NOT NULL DEFAULT '';
ALTER TABLE commerce_customer_orders ADD COLUMN IF NOT EXISTS invoice_claimed_at TIMESTAMPTZ;
ALTER TABLE commerce_customer_orders ADD COLUMN IF NOT EXISTS invoiced_at TIMESTAMPTZ;
CREATE INDEX IF NOT EXISTS idx_commerce_customer_orders_invoice_due ON commerce_customer_orders(tenant_id, payment_status, invoice_status);
