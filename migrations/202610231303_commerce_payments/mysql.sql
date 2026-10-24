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
    id VARCHAR(128) PRIMARY KEY,
    order_id VARCHAR(128) NOT NULL,
    provider VARCHAR(64) NOT NULL,
    provider_payment_id VARCHAR(128) NOT NULL,
    amount_cents INTEGER NOT NULL CHECK (amount_cents >= 0),
    currency VARCHAR(8) NOT NULL,
    status VARCHAR(32) NOT NULL,
    created_at DATETIME(6) NOT NULL,
    updated_at DATETIME(6) NOT NULL,
    tenant_id VARCHAR(128) NOT NULL DEFAULT '',
    UNIQUE KEY idx_commerce_payment_charges_provider (provider, provider_payment_id),
    INDEX idx_commerce_payment_charges_order (order_id),
    CONSTRAINT commerce_payment_charges_order_fk FOREIGN KEY (order_id) REFERENCES commerce_customer_orders(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS commerce_payment_notifications (
    provider VARCHAR(64) NOT NULL,
    provider_event_id VARCHAR(128) NOT NULL,
    received_at DATETIME(6) NOT NULL,
    tenant_id VARCHAR(128) NOT NULL DEFAULT '',
    PRIMARY KEY (provider, provider_event_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
