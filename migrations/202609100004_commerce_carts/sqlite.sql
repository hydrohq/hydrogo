CREATE TABLE commerce_carts (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL UNIQUE,
    currency TEXT NOT NULL,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);
CREATE TABLE commerce_cart_items (
    id TEXT PRIMARY KEY,
    cart_id TEXT NOT NULL REFERENCES commerce_carts(id) ON DELETE CASCADE,
    product_id TEXT NOT NULL REFERENCES commerce_products(id),
    quantity INTEGER NOT NULL CHECK (quantity > 0),
    unit_cents INTEGER NOT NULL CHECK (unit_cents >= 0),
    UNIQUE(cart_id, product_id)
);
