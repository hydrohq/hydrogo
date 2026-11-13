ALTER TABLE commerce_customer_orders
    ADD COLUMN shipping_shipment_id VARCHAR(128) NOT NULL DEFAULT '',
    ADD COLUMN shipping_label_url VARCHAR(1024) NOT NULL DEFAULT '',
    ADD COLUMN delivered_at DATETIME(6),
    ADD INDEX idx_commerce_customer_orders_shipment (tenant_id, shipping_shipment_id);
