ALTER TABLE commerce_products ADD COLUMN tenant_id VARCHAR(128) NOT NULL DEFAULT '';
ALTER TABLE commerce_product_variants ADD COLUMN tenant_id VARCHAR(128) NOT NULL DEFAULT '';
CREATE INDEX idx_commerce_products_tenant_status ON commerce_products(tenant_id, status, title(400), id);
CREATE INDEX idx_commerce_variants_tenant_product ON commerce_product_variants(tenant_id, product_id, status, title(300), id);
