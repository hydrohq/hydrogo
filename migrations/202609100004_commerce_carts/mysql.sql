CREATE TABLE commerce_carts (
    id VARCHAR(128) PRIMARY KEY,
    user_id VARCHAR(128) NOT NULL UNIQUE,
    currency CHAR(3) NOT NULL,
    created_at DATETIME(6) NOT NULL,
    updated_at DATETIME(6) NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE commerce_cart_items (
    id VARCHAR(128) PRIMARY KEY,
    cart_id VARCHAR(128) NOT NULL,
    product_id VARCHAR(128) NOT NULL,
    quantity BIGINT NOT NULL CHECK (quantity > 0),
    unit_cents BIGINT NOT NULL CHECK (unit_cents >= 0),
    UNIQUE (cart_id, product_id),
    CONSTRAINT commerce_cart_items_cart_fk FOREIGN KEY (cart_id) REFERENCES commerce_carts(id) ON DELETE CASCADE,
    CONSTRAINT commerce_cart_items_product_fk FOREIGN KEY (product_id) REFERENCES commerce_products(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
