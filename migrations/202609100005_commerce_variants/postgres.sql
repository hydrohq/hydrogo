CREATE TABLE commerce_product_variants (
    id TEXT PRIMARY KEY,
    product_id TEXT NOT NULL REFERENCES commerce_products(id) ON DELETE CASCADE,
    sku TEXT NOT NULL UNIQUE,
    title TEXT NOT NULL,
    price_cents BIGINT NOT NULL CHECK (price_cents >= 0),
    stock INTEGER NOT NULL CHECK (stock >= 0),
    status TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL
);
CREATE INDEX idx_commerce_variants_product ON commerce_product_variants(product_id, title, id);
ALTER TABLE commerce_cart_items ADD COLUMN variant_id TEXT REFERENCES commerce_product_variants(id);
ALTER TABLE commerce_order_items ADD COLUMN variant_id TEXT REFERENCES commerce_product_variants(id);
