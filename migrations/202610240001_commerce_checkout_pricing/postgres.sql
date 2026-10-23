ALTER TABLE commerce_customer_orders ADD COLUMN IF NOT EXISTS subtotal_cents INTEGER NOT NULL DEFAULT 0;
ALTER TABLE commerce_customer_orders ADD COLUMN IF NOT EXISTS discount_code TEXT NOT NULL DEFAULT '';
ALTER TABLE commerce_customer_orders ADD COLUMN IF NOT EXISTS discount_cents INTEGER NOT NULL DEFAULT 0;
ALTER TABLE commerce_customer_orders ADD COLUMN IF NOT EXISTS tax_cents INTEGER NOT NULL DEFAULT 0;
ALTER TABLE commerce_customer_orders ADD COLUMN IF NOT EXISTS shipping_cents INTEGER NOT NULL DEFAULT 0;

CREATE TABLE IF NOT EXISTS commerce_discount_codes (
    id TEXT PRIMARY KEY,
    code TEXT NOT NULL,
    kind TEXT NOT NULL,
    value INTEGER NOT NULL CHECK (value >= 0),
    min_subtotal_cents INTEGER NOT NULL DEFAULT 0 CHECK (min_subtotal_cents >= 0),
    max_redemptions INTEGER NOT NULL DEFAULT 0 CHECK (max_redemptions >= 0),
    redeemed_count INTEGER NOT NULL DEFAULT 0 CHECK (redeemed_count >= 0),
    active BOOLEAN NOT NULL DEFAULT TRUE,
    starts_at TIMESTAMPTZ,
    ends_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,
    tenant_id TEXT NOT NULL DEFAULT '',
    UNIQUE (tenant_id, code)
);

ALTER TABLE commerce_discount_codes ENABLE ROW LEVEL SECURITY;
ALTER TABLE commerce_discount_codes FORCE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS hydrogo_tenant_isolation ON commerce_discount_codes;
CREATE POLICY hydrogo_tenant_isolation ON commerce_discount_codes
    USING ((coalesce(nullif(current_setting('app.tenant_rls_enforced', true), ''), 'false') <> 'true') OR tenant_id = nullif(current_setting('app.current_tenant_id', true), ''))
    WITH CHECK ((coalesce(nullif(current_setting('app.tenant_rls_enforced', true), ''), 'false') <> 'true') OR tenant_id = nullif(current_setting('app.current_tenant_id', true), ''));
