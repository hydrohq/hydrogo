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
-- nullable column is enough to scope each kind of identity: Postgres,
-- like standard SQL, does not treat multiple NULLs as duplicates.
CREATE TABLE IF NOT EXISTS commerce_shopping_carts (
    id TEXT PRIMARY KEY,
    member_id TEXT,
    anonymous_token TEXT,
    currency TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    tenant_id TEXT NOT NULL DEFAULT '',
    CHECK ((member_id IS NULL) != (anonymous_token IS NULL))
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_commerce_shopping_carts_member ON commerce_shopping_carts(tenant_id, member_id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_commerce_shopping_carts_anonymous ON commerce_shopping_carts(tenant_id, anonymous_token);
CREATE INDEX IF NOT EXISTS idx_commerce_shopping_carts_expiry ON commerce_shopping_carts(expires_at);

CREATE TABLE IF NOT EXISTS commerce_shopping_cart_items (
    id TEXT PRIMARY KEY,
    cart_id TEXT NOT NULL REFERENCES commerce_shopping_carts(id) ON DELETE CASCADE,
    variant_id TEXT NOT NULL,
    quantity INTEGER NOT NULL CHECK (quantity > 0),
    unit_price_cents INTEGER NOT NULL CHECK (unit_price_cents >= 0),
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,
    tenant_id TEXT NOT NULL DEFAULT '',
    UNIQUE (cart_id, variant_id)
);
CREATE INDEX IF NOT EXISTS idx_commerce_shopping_cart_items_cart ON commerce_shopping_cart_items(cart_id);

ALTER TABLE commerce_shopping_carts ENABLE ROW LEVEL SECURITY;
ALTER TABLE commerce_shopping_carts FORCE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS hydrogo_tenant_isolation ON commerce_shopping_carts;
CREATE POLICY hydrogo_tenant_isolation ON commerce_shopping_carts
    USING ((coalesce(nullif(current_setting('app.tenant_rls_enforced', true), ''), 'false') <> 'true') OR tenant_id = nullif(current_setting('app.current_tenant_id', true), ''))
    WITH CHECK ((coalesce(nullif(current_setting('app.tenant_rls_enforced', true), ''), 'false') <> 'true') OR tenant_id = nullif(current_setting('app.current_tenant_id', true), ''));

ALTER TABLE commerce_shopping_cart_items ENABLE ROW LEVEL SECURITY;
ALTER TABLE commerce_shopping_cart_items FORCE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS hydrogo_tenant_isolation ON commerce_shopping_cart_items;
CREATE POLICY hydrogo_tenant_isolation ON commerce_shopping_cart_items
    USING ((coalesce(nullif(current_setting('app.tenant_rls_enforced', true), ''), 'false') <> 'true') OR tenant_id = nullif(current_setting('app.current_tenant_id', true), ''))
    WITH CHECK ((coalesce(nullif(current_setting('app.tenant_rls_enforced', true), ''), 'false') <> 'true') OR tenant_id = nullif(current_setting('app.current_tenant_id', true), ''));
