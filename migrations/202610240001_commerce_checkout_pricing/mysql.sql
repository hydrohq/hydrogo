ALTER TABLE commerce_customer_orders ADD COLUMN subtotal_cents INTEGER NOT NULL DEFAULT 0;
ALTER TABLE commerce_customer_orders ADD COLUMN discount_code VARCHAR(64) NOT NULL DEFAULT '';
ALTER TABLE commerce_customer_orders ADD COLUMN discount_cents INTEGER NOT NULL DEFAULT 0;
ALTER TABLE commerce_customer_orders ADD COLUMN tax_cents INTEGER NOT NULL DEFAULT 0;
ALTER TABLE commerce_customer_orders ADD COLUMN shipping_cents INTEGER NOT NULL DEFAULT 0;

CREATE TABLE IF NOT EXISTS commerce_discount_codes (
    id VARCHAR(128) PRIMARY KEY,
    code VARCHAR(64) NOT NULL,
    kind VARCHAR(16) NOT NULL,
    value INTEGER NOT NULL CHECK (value >= 0),
    min_subtotal_cents INTEGER NOT NULL DEFAULT 0 CHECK (min_subtotal_cents >= 0),
    max_redemptions INTEGER NOT NULL DEFAULT 0 CHECK (max_redemptions >= 0),
    redeemed_count INTEGER NOT NULL DEFAULT 0 CHECK (redeemed_count >= 0),
    active TINYINT(1) NOT NULL DEFAULT 1,
    starts_at DATETIME(6),
    ends_at DATETIME(6),
    created_at DATETIME(6) NOT NULL,
    updated_at DATETIME(6) NOT NULL,
    tenant_id VARCHAR(128) NOT NULL DEFAULT '',
    UNIQUE KEY idx_commerce_discount_codes_code (tenant_id, code)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
