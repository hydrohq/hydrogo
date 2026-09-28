CREATE TABLE commerce_products (
    id VARCHAR(128) PRIMARY KEY,
    sku VARCHAR(191) NOT NULL UNIQUE,
    title VARCHAR(512) NOT NULL,
    price_cents BIGINT NOT NULL CHECK (price_cents >= 0),
    currency CHAR(3) NOT NULL,
    stock BIGINT NOT NULL CHECK (stock >= 0),
    status VARCHAR(24) NOT NULL,
    created_at DATETIME(6) NOT NULL,
    updated_at DATETIME(6) NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE commerce_orders (
    id VARCHAR(128) PRIMARY KEY,
    user_id VARCHAR(128) NOT NULL,
    idempotency_key VARCHAR(128) NOT NULL,
    currency CHAR(3) NOT NULL,
    total_cents BIGINT NOT NULL CHECK (total_cents >= 0),
    payment_status VARCHAR(24) NOT NULL,
    shipping_status VARCHAR(24) NOT NULL,
    created_at DATETIME(6) NOT NULL,
    UNIQUE (user_id, idempotency_key)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE commerce_order_items (
    id VARCHAR(128) PRIMARY KEY,
    order_id VARCHAR(128) NOT NULL,
    product_id VARCHAR(128) NOT NULL,
    sku VARCHAR(191) NOT NULL,
    title VARCHAR(512) NOT NULL,
    quantity BIGINT NOT NULL CHECK (quantity > 0),
    unit_cents BIGINT NOT NULL CHECK (unit_cents >= 0),
    total_cents BIGINT NOT NULL CHECK (total_cents >= 0),
    CONSTRAINT commerce_order_items_order_fk FOREIGN KEY (order_id) REFERENCES commerce_orders(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
