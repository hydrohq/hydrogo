CREATE TABLE IF NOT EXISTS plans (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    code TEXT NOT NULL UNIQUE,
    description TEXT NOT NULL DEFAULT '',
    price_cents BIGINT NOT NULL,
    currency VARCHAR(3) NOT NULL DEFAULT 'TRY',
    billing_interval TEXT NOT NULL,
    interval_count INT NOT NULL DEFAULT 1,
    trial_days INT NOT NULL DEFAULT 0,
    features TEXT NOT NULL DEFAULT '[]',
    active BOOLEAN NOT NULL DEFAULT TRUE,
    tenant_id TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_plans_code ON plans(code);
CREATE INDEX IF NOT EXISTS idx_plans_active ON plans(active);

CREATE TABLE IF NOT EXISTS orders (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    plan_id TEXT NOT NULL REFERENCES plans(id),
    order_number TEXT NOT NULL UNIQUE,
    status TEXT NOT NULL,
    amount_cents BIGINT NOT NULL,
    currency VARCHAR(3) NOT NULL DEFAULT 'TRY',
    billing_interval TEXT NOT NULL,
    idempotency_key TEXT UNIQUE,
    notes TEXT NOT NULL DEFAULT '',
    tenant_id TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_orders_user_id ON orders(user_id);
CREATE INDEX IF NOT EXISTS idx_orders_order_number ON orders(order_number);
CREATE INDEX IF NOT EXISTS idx_orders_status ON orders(status);
CREATE INDEX IF NOT EXISTS idx_orders_idempotency_key ON orders(idempotency_key);

CREATE TABLE IF NOT EXISTS payments (
    id TEXT PRIMARY KEY,
    order_id TEXT NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    payment_provider TEXT NOT NULL DEFAULT 'shopier',
    provider_payment_id TEXT NOT NULL DEFAULT '',
    amount_cents BIGINT NOT NULL,
    currency VARCHAR(3) NOT NULL DEFAULT 'TRY',
    status TEXT NOT NULL,
    idempotency_key TEXT UNIQUE,
    error_message TEXT NOT NULL DEFAULT '',
    metadata TEXT NOT NULL DEFAULT '{}',
    paid_at TIMESTAMPTZ,
    tenant_id TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_payments_order_id ON payments(order_id);
CREATE INDEX IF NOT EXISTS idx_payments_user_id ON payments(user_id);
CREATE INDEX IF NOT EXISTS idx_payments_status ON payments(status);
CREATE INDEX IF NOT EXISTS idx_payments_provider_id ON payments(provider_payment_id);

CREATE TABLE IF NOT EXISTS subscriptions (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    plan_id TEXT NOT NULL REFERENCES plans(id),
    status TEXT NOT NULL,
    current_period_start TIMESTAMPTZ NOT NULL,
    current_period_end TIMESTAMPTZ NOT NULL,
    cancel_at_period_end BOOLEAN NOT NULL DEFAULT FALSE,
    cancelled_at TIMESTAMPTZ,
    tenant_id TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_subscriptions_user_id ON subscriptions(user_id);
CREATE INDEX IF NOT EXISTS idx_subscriptions_status ON subscriptions(status);
CREATE INDEX IF NOT EXISTS idx_subscriptions_period_end ON subscriptions(current_period_end);

CREATE TABLE IF NOT EXISTS payment_events (
    id TEXT PRIMARY KEY,
    payment_id TEXT REFERENCES payments(id) ON DELETE SET NULL,
    event_type TEXT NOT NULL,
    provider TEXT NOT NULL DEFAULT 'shopier',
    provider_reference TEXT NOT NULL DEFAULT '',
    signature TEXT NOT NULL DEFAULT '',
    payload TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL DEFAULT 'processed',
    tenant_id TEXT NOT NULL DEFAULT '',
    processed_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    CONSTRAINT uq_payment_events_provider_ref UNIQUE (provider, provider_reference, event_type)
);

CREATE INDEX IF NOT EXISTS idx_payment_events_ref ON payment_events(provider, provider_reference);

INSERT INTO plans (id, name, code, description, price_cents, currency, billing_interval, interval_count, features, active, tenant_id, created_at, updated_at)
VALUES
    ('plan_monthly', 'HydroCloud Pro (Aylık)', 'monthly', 'Tam donanımlı mağaza ve SaaS altyapısı - Aylık abonelik', 49900, 'TRY', 'month', 1, '["Sınırsız Ürün ve Sayfa", "Shopier Güvenli Ödeme", "Özel Alan Adı Bağlama", "Yapay Zeka Destekli Copilot", "7/24 Teknik Destek"]', TRUE, '', '2026-11-08T00:00:00Z', '2026-11-08T00:00:00Z'),
    ('plan_yearly', 'HydroCloud Pro (Yıllık)', 'yearly', 'Tam donanımlı mağaza ve SaaS altyapısı - Yıllık abonelik (2 Ay Ücretsiz)', 499000, 'TRY', 'year', 1, '["Sınırsız Ürün ve Sayfa", "Shopier Güvenli Ödeme", "Özel Alan Adı Bağlama", "Yapay Zeka Destekli Copilot", "Öncelikli 7/24 VIP Destek", "Yıllık Ödemede 2 Ay Hediye"]', TRUE, '', '2026-11-08T00:00:00Z', '2026-11-08T00:00:00Z')
ON CONFLICT (id) DO NOTHING;
