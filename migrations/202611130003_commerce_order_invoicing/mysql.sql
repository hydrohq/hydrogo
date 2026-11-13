ALTER TABLE commerce_customer_orders
    ADD COLUMN invoice_status VARCHAR(32) NOT NULL DEFAULT 'none',
    ADD COLUMN invoice_provider VARCHAR(64) NOT NULL DEFAULT '',
    ADD COLUMN invoice_guid VARCHAR(128) NOT NULL DEFAULT '',
    ADD COLUMN invoice_no VARCHAR(128) NOT NULL DEFAULT '',
    ADD COLUMN invoice_url VARCHAR(1024) NOT NULL DEFAULT '',
    ADD COLUMN invoice_error VARCHAR(1024) NOT NULL DEFAULT '',
    ADD COLUMN invoice_claimed_at DATETIME(6),
    ADD COLUMN invoiced_at DATETIME(6),
    ADD INDEX idx_commerce_customer_orders_invoice_due (tenant_id, payment_status, invoice_status);
