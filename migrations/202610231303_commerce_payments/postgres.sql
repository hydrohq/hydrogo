-- Named commerce_payment_charges / commerce_payment_notifications, not
-- commerce_payment_transactions / commerce_payment_webhook_events: the old,
-- removed commerce module's migrations (still on disk per the never-delete
-- rule, and still baselined in migrations/baseline/commerce/) already own
-- those names. A fresh database enabling the commerce module applies that
-- old baseline group too (its 11 migration IDs are all pending on a new
-- install), so reusing its table names here would silently collide with it.
--
-- commerce_payment_notifications is the dedup table for provider callbacks
-- (PayTR, and any future provider): a provider may redeliver the same
-- notification more than once by design, and the primary key here is what
-- makes processing it idempotent -- see modules/commerce/providers.
CREATE TABLE IF NOT EXISTS commerce_payment_charges (
    id TEXT PRIMARY KEY,
    order_id TEXT NOT NULL REFERENCES commerce_customer_orders(id) ON DELETE CASCADE,
    provider TEXT NOT NULL,
    provider_payment_id TEXT NOT NULL,
    amount_cents INTEGER NOT NULL CHECK (amount_cents >= 0),
    currency TEXT NOT NULL,
    status TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,
    tenant_id TEXT NOT NULL DEFAULT '',
    UNIQUE (provider, provider_payment_id)
);
CREATE INDEX IF NOT EXISTS idx_commerce_payment_charges_order ON commerce_payment_charges(order_id);

CREATE TABLE IF NOT EXISTS commerce_payment_notifications (
    provider TEXT NOT NULL,
    provider_event_id TEXT NOT NULL,
    received_at TIMESTAMPTZ NOT NULL,
    tenant_id TEXT NOT NULL DEFAULT '',
    PRIMARY KEY (provider, provider_event_id)
);

ALTER TABLE commerce_payment_charges ENABLE ROW LEVEL SECURITY;
ALTER TABLE commerce_payment_charges FORCE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS hydrogo_tenant_isolation ON commerce_payment_charges;
CREATE POLICY hydrogo_tenant_isolation ON commerce_payment_charges
    USING ((coalesce(nullif(current_setting('app.tenant_rls_enforced', true), ''), 'false') <> 'true') OR tenant_id = nullif(current_setting('app.current_tenant_id', true), ''))
    WITH CHECK ((coalesce(nullif(current_setting('app.tenant_rls_enforced', true), ''), 'false') <> 'true') OR tenant_id = nullif(current_setting('app.current_tenant_id', true), ''));

ALTER TABLE commerce_payment_notifications ENABLE ROW LEVEL SECURITY;
ALTER TABLE commerce_payment_notifications FORCE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS hydrogo_tenant_isolation ON commerce_payment_notifications;
CREATE POLICY hydrogo_tenant_isolation ON commerce_payment_notifications
    USING ((coalesce(nullif(current_setting('app.tenant_rls_enforced', true), ''), 'false') <> 'true') OR tenant_id = nullif(current_setting('app.current_tenant_id', true), ''))
    WITH CHECK ((coalesce(nullif(current_setting('app.tenant_rls_enforced', true), ''), 'false') <> 'true') OR tenant_id = nullif(current_setting('app.current_tenant_id', true), ''));
