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
    id TEXT PRIMARY KEY,
    member_id TEXT,
    guest_name TEXT NOT NULL DEFAULT '',
    guest_email TEXT NOT NULL DEFAULT '',
    guest_phone TEXT NOT NULL DEFAULT '',
    guest_address TEXT NOT NULL DEFAULT '',
    guest_city TEXT NOT NULL DEFAULT '',
    guest_country TEXT NOT NULL DEFAULT '',
    guest_postal_code TEXT NOT NULL DEFAULT '',
    idempotency_key TEXT NOT NULL,
    idempotency_hash TEXT NOT NULL,
    currency TEXT NOT NULL,
    total_cents INTEGER NOT NULL CHECK (total_cents >= 0),
    payment_status TEXT NOT NULL DEFAULT 'pending',
    fulfillment_status TEXT NOT NULL DEFAULT 'unfulfilled',
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,
    tenant_id TEXT NOT NULL DEFAULT '',
    UNIQUE (tenant_id, idempotency_key)
);
CREATE INDEX IF NOT EXISTS idx_commerce_customer_orders_member ON commerce_customer_orders(tenant_id, member_id);
CREATE INDEX IF NOT EXISTS idx_commerce_customer_orders_status ON commerce_customer_orders(tenant_id, payment_status, fulfillment_status);

CREATE TABLE IF NOT EXISTS commerce_customer_order_items (
    id TEXT PRIMARY KEY,
    order_id TEXT NOT NULL REFERENCES commerce_customer_orders(id) ON DELETE CASCADE,
    variant_id TEXT NOT NULL,
    sku TEXT NOT NULL,
    title TEXT NOT NULL,
    quantity INTEGER NOT NULL CHECK (quantity > 0),
    unit_price_cents INTEGER NOT NULL CHECK (unit_price_cents >= 0),
    total_cents INTEGER NOT NULL CHECK (total_cents >= 0),
    created_at TIMESTAMPTZ NOT NULL,
    tenant_id TEXT NOT NULL DEFAULT ''
);
CREATE INDEX IF NOT EXISTS idx_commerce_customer_order_items_order ON commerce_customer_order_items(order_id);

ALTER TABLE commerce_customer_orders ENABLE ROW LEVEL SECURITY;
ALTER TABLE commerce_customer_orders FORCE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS hydrogo_tenant_isolation ON commerce_customer_orders;
CREATE POLICY hydrogo_tenant_isolation ON commerce_customer_orders
    USING ((coalesce(nullif(current_setting('app.tenant_rls_enforced', true), ''), 'false') <> 'true') OR tenant_id = nullif(current_setting('app.current_tenant_id', true), ''))
    WITH CHECK ((coalesce(nullif(current_setting('app.tenant_rls_enforced', true), ''), 'false') <> 'true') OR tenant_id = nullif(current_setting('app.current_tenant_id', true), ''));

ALTER TABLE commerce_customer_order_items ENABLE ROW LEVEL SECURITY;
ALTER TABLE commerce_customer_order_items FORCE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS hydrogo_tenant_isolation ON commerce_customer_order_items;
CREATE POLICY hydrogo_tenant_isolation ON commerce_customer_order_items
    USING ((coalesce(nullif(current_setting('app.tenant_rls_enforced', true), ''), 'false') <> 'true') OR tenant_id = nullif(current_setting('app.current_tenant_id', true), ''))
    WITH CHECK ((coalesce(nullif(current_setting('app.tenant_rls_enforced', true), ''), 'false') <> 'true') OR tenant_id = nullif(current_setting('app.current_tenant_id', true), ''));
