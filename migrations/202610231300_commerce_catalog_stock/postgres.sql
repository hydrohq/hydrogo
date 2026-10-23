-- Stock lives here, not on a content-type record: selling the last unit
-- needs an atomic "decrement if enough is available" update, which the
-- generic collections store (optimistic version-locking only) can't give.
-- variant_id is a content-type record id (content-types/commerce/variants.yaml),
-- not a foreign key: content records live in the generic
-- hydro_collection_records table, which can't be referenced across dialects.
CREATE TABLE IF NOT EXISTS commerce_stock (
    id TEXT PRIMARY KEY,
    variant_id TEXT NOT NULL,
    available INTEGER NOT NULL DEFAULT 0 CHECK (available >= 0),
    reserved INTEGER NOT NULL DEFAULT 0 CHECK (reserved >= 0),
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,
    tenant_id TEXT NOT NULL DEFAULT ''
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_commerce_stock_variant ON commerce_stock(tenant_id, variant_id);

ALTER TABLE commerce_stock ENABLE ROW LEVEL SECURITY;
ALTER TABLE commerce_stock FORCE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS hydrogo_tenant_isolation ON commerce_stock;
CREATE POLICY hydrogo_tenant_isolation ON commerce_stock
    USING ((coalesce(nullif(current_setting('app.tenant_rls_enforced', true), ''), 'false') <> 'true') OR tenant_id = nullif(current_setting('app.current_tenant_id', true), ''))
    WITH CHECK ((coalesce(nullif(current_setting('app.tenant_rls_enforced', true), ''), 'false') <> 'true') OR tenant_id = nullif(current_setting('app.current_tenant_id', true), ''));
