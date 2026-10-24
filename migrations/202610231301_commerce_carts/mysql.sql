-- Named commerce_shopping_carts, not commerce_carts: the old, removed
-- commerce module's migrations (still on disk per the never-delete rule,
-- and still baselined in migrations/baseline/commerce/) already own the
-- name commerce_carts for a materially different, member-only schema. A
-- fresh database enabling the commerce module applies that old baseline
-- group too (its 11 migration IDs are all pending on a new install), so
-- reusing its table names here would silently collide with it.
--
-- A cart belongs to exactly one of a logged-in member or an anonymous
-- visitor (identified by an opaque, signed cookie token minted by
-- modules/commerce) -- never both, never neither. Plain UNIQUE on a
-- nullable column is enough to scope each kind of identity: MySQL,
-- like standard SQL, does not treat multiple NULLs as duplicates.
CREATE TABLE IF NOT EXISTS commerce_shopping_carts (
    id VARCHAR(128) PRIMARY KEY,
    member_id VARCHAR(128),
    anonymous_token VARCHAR(128),
    currency VARCHAR(8) NOT NULL,
    created_at DATETIME(6) NOT NULL,
    updated_at DATETIME(6) NOT NULL,
    expires_at DATETIME(6) NOT NULL,
    tenant_id VARCHAR(128) NOT NULL DEFAULT '',
    CHECK ((member_id IS NULL) != (anonymous_token IS NULL)),
    UNIQUE KEY idx_commerce_shopping_carts_member (tenant_id, member_id),
    UNIQUE KEY idx_commerce_shopping_carts_anonymous (tenant_id, anonymous_token),
    INDEX idx_commerce_shopping_carts_expiry (expires_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS commerce_shopping_cart_items (
    id VARCHAR(128) PRIMARY KEY,
    cart_id VARCHAR(128) NOT NULL,
    variant_id VARCHAR(128) NOT NULL,
    quantity INTEGER NOT NULL CHECK (quantity > 0),
    unit_price_cents INTEGER NOT NULL CHECK (unit_price_cents >= 0),
    created_at DATETIME(6) NOT NULL,
    updated_at DATETIME(6) NOT NULL,
    tenant_id VARCHAR(128) NOT NULL DEFAULT '',
    UNIQUE KEY idx_commerce_shopping_cart_items_variant (cart_id, variant_id),
    INDEX idx_commerce_shopping_cart_items_cart (cart_id),
    CONSTRAINT commerce_shopping_cart_items_cart_fk FOREIGN KEY (cart_id) REFERENCES commerce_shopping_carts(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
