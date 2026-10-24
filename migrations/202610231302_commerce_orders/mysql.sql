-- Named commerce_customer_orders, not commerce_orders: the old, removed
-- commerce module's migrations (still on disk per the never-delete rule,
-- and still baselined in migrations/baseline/commerce/) already own the
-- name commerce_orders. A fresh database enabling the commerce module
-- applies that old baseline group too (its 11 migration IDs are all
-- pending on a new install), so reusing its table names here would
-- silently collide with it.
--
-- Guest fields are always present (unlike the old module, where an order
-- could carry guest contact details but the cart itself could only belong
-- to a logged-in member). idempotency_key is scoped by tenant alone, not
-- by member_id: a guest checkout has no member_id to scope by, and NULLs
-- are never equal to each other in a UNIQUE index, which would silently
-- defeat idempotency for every guest order.
CREATE TABLE IF NOT EXISTS commerce_customer_orders (
    id VARCHAR(128) PRIMARY KEY,
    member_id VARCHAR(128),
    guest_name VARCHAR(255) NOT NULL DEFAULT '',
    guest_email VARCHAR(255) NOT NULL DEFAULT '',
    guest_phone VARCHAR(64) NOT NULL DEFAULT '',
    guest_address VARCHAR(500) NOT NULL DEFAULT '',
    guest_city VARCHAR(128) NOT NULL DEFAULT '',
    guest_country VARCHAR(128) NOT NULL DEFAULT '',
    guest_postal_code VARCHAR(32) NOT NULL DEFAULT '',
    idempotency_key VARCHAR(128) NOT NULL,
    idempotency_hash VARCHAR(128) NOT NULL,
    currency VARCHAR(8) NOT NULL,
    total_cents INTEGER NOT NULL CHECK (total_cents >= 0),
    payment_status VARCHAR(32) NOT NULL DEFAULT 'pending',
    fulfillment_status VARCHAR(32) NOT NULL DEFAULT 'unfulfilled',
    created_at DATETIME(6) NOT NULL,
    updated_at DATETIME(6) NOT NULL,
    tenant_id VARCHAR(128) NOT NULL DEFAULT '',
    UNIQUE KEY idx_commerce_customer_orders_idempotency (tenant_id, idempotency_key),
    INDEX idx_commerce_customer_orders_member (tenant_id, member_id),
    INDEX idx_commerce_customer_orders_status (tenant_id, payment_status, fulfillment_status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS commerce_customer_order_items (
    id VARCHAR(128) PRIMARY KEY,
    order_id VARCHAR(128) NOT NULL,
    variant_id VARCHAR(128) NOT NULL,
    sku VARCHAR(128) NOT NULL,
    title VARCHAR(255) NOT NULL,
    quantity INTEGER NOT NULL CHECK (quantity > 0),
    unit_price_cents INTEGER NOT NULL CHECK (unit_price_cents >= 0),
    total_cents INTEGER NOT NULL CHECK (total_cents >= 0),
    created_at DATETIME(6) NOT NULL,
    tenant_id VARCHAR(128) NOT NULL DEFAULT '',
    INDEX idx_commerce_customer_order_items_order (order_id),
    CONSTRAINT commerce_customer_order_items_order_fk FOREIGN KEY (order_id) REFERENCES commerce_customer_orders(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
