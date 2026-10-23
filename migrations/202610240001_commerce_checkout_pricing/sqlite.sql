-- Added after commerce_customer_orders already shipped (202610231302), so
-- this is an ALTER, not a rewrite of that table's CREATE statement -- see
-- AGENTS.md rule 3, never edit an already-released migration.
ALTER TABLE commerce_customer_orders ADD COLUMN subtotal_cents INTEGER NOT NULL DEFAULT 0;
ALTER TABLE commerce_customer_orders ADD COLUMN discount_code TEXT NOT NULL DEFAULT '';
ALTER TABLE commerce_customer_orders ADD COLUMN discount_cents INTEGER NOT NULL DEFAULT 0;
ALTER TABLE commerce_customer_orders ADD COLUMN tax_cents INTEGER NOT NULL DEFAULT 0;
ALTER TABLE commerce_customer_orders ADD COLUMN shipping_cents INTEGER NOT NULL DEFAULT 0;

-- A discount code's redemption count is incremented by one conditional
-- UPDATE (WHERE max_redemptions = 0 OR redeemed_count < max_redemptions),
-- the same atomic-guard pattern commerce_stock's ReserveStock already uses
-- for the same reason: two checkouts racing the last redemption must never
-- both succeed.
CREATE TABLE IF NOT EXISTS commerce_discount_codes (
    id TEXT PRIMARY KEY,
    code TEXT NOT NULL,
    kind TEXT NOT NULL,
    value INTEGER NOT NULL CHECK (value >= 0),
    min_subtotal_cents INTEGER NOT NULL DEFAULT 0 CHECK (min_subtotal_cents >= 0),
    max_redemptions INTEGER NOT NULL DEFAULT 0 CHECK (max_redemptions >= 0),
    redeemed_count INTEGER NOT NULL DEFAULT 0 CHECK (redeemed_count >= 0),
    active INTEGER NOT NULL DEFAULT 1,
    starts_at TEXT,
    ends_at TEXT,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    tenant_id TEXT NOT NULL DEFAULT '',
    UNIQUE (tenant_id, code)
);
