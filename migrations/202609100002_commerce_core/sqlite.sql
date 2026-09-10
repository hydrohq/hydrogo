CREATE TABLE commerce_products (
    id TEXT PRIMARY KEY,
    sku TEXT NOT NULL UNIQUE,
    title TEXT NOT NULL,
    price_cents INTEGER NOT NULL CHECK (price_cents >= 0),
    currency TEXT NOT NULL,
    stock INTEGER NOT NULL CHECK (stock >= 0),
    status TEXT NOT NULL,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);
CREATE TABLE commerce_orders (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    idempotency_key TEXT NOT NULL,
    currency TEXT NOT NULL,
    total_cents INTEGER NOT NULL CHECK (total_cents >= 0),
    payment_status TEXT NOT NULL,
    shipping_status TEXT NOT NULL,
    created_at TEXT NOT NULL,
    UNIQUE(user_id, idempotency_key)
);
CREATE TABLE commerce_order_items (
    id TEXT PRIMARY KEY,
    order_id TEXT NOT NULL REFERENCES commerce_orders(id) ON DELETE CASCADE,
    product_id TEXT NOT NULL,
    sku TEXT NOT NULL,
    title TEXT NOT NULL,
    quantity INTEGER NOT NULL CHECK (quantity > 0),
    unit_cents INTEGER NOT NULL CHECK (unit_cents >= 0),
    total_cents INTEGER NOT NULL CHECK (total_cents >= 0)
);
