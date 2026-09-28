CREATE TABLE commerce_product_variants (
    id VARCHAR(128) PRIMARY KEY,
    product_id VARCHAR(128) NOT NULL,
    sku VARCHAR(191) NOT NULL UNIQUE,
    title VARCHAR(512) NOT NULL,
    price_cents BIGINT NOT NULL CHECK (price_cents >= 0),
    stock BIGINT NOT NULL CHECK (stock >= 0),
    status VARCHAR(24) NOT NULL,
    created_at DATETIME(6) NOT NULL,
    updated_at DATETIME(6) NOT NULL,
    INDEX idx_commerce_variants_product (product_id, title, id),
    CONSTRAINT commerce_variants_product_fk FOREIGN KEY (product_id) REFERENCES commerce_products(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

ALTER TABLE commerce_cart_items ADD COLUMN variant_id VARCHAR(128);
ALTER TABLE commerce_cart_items ADD CONSTRAINT commerce_cart_variant_fk FOREIGN KEY (variant_id) REFERENCES commerce_product_variants(id);
ALTER TABLE commerce_order_items ADD COLUMN variant_id VARCHAR(128);
ALTER TABLE commerce_order_items ADD CONSTRAINT commerce_order_variant_fk FOREIGN KEY (variant_id) REFERENCES commerce_product_variants(id);
