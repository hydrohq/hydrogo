CREATE TABLE IF NOT EXISTS `plans` (
    `id` VARCHAR(128) PRIMARY KEY,
    `name` VARCHAR(255) NOT NULL,
    `code` VARCHAR(64) NOT NULL UNIQUE,
    `description` TEXT NOT NULL,
    `price_cents` BIGINT NOT NULL,
    `currency` VARCHAR(3) NOT NULL DEFAULT 'TRY',
    `billing_interval` VARCHAR(32) NOT NULL,
    `interval_count` INT NOT NULL DEFAULT 1,
    `trial_days` INT NOT NULL DEFAULT 0,
    `features` TEXT NOT NULL,
    `active` TINYINT(1) NOT NULL DEFAULT 1,
    `tenant_id` VARCHAR(128) NOT NULL DEFAULT '',
    `created_at` DATETIME(6) NOT NULL,
    `updated_at` DATETIME(6) NOT NULL,
    INDEX `idx_plans_code` (`code`),
    INDEX `idx_plans_active` (`active`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS `orders` (
    `id` VARCHAR(128) PRIMARY KEY,
    `user_id` VARCHAR(128) NOT NULL,
    `plan_id` VARCHAR(128) NOT NULL,
    `order_number` VARCHAR(64) NOT NULL UNIQUE,
    `status` VARCHAR(32) NOT NULL,
    `amount_cents` BIGINT NOT NULL,
    `currency` VARCHAR(3) NOT NULL DEFAULT 'TRY',
    `billing_interval` VARCHAR(32) NOT NULL,
    `idempotency_key` VARCHAR(128) UNIQUE,
    `notes` TEXT NOT NULL,
    `tenant_id` VARCHAR(128) NOT NULL DEFAULT '',
    `created_at` DATETIME(6) NOT NULL,
    `updated_at` DATETIME(6) NOT NULL,
    INDEX `idx_orders_user_id` (`user_id`),
    INDEX `idx_orders_order_number` (`order_number`),
    INDEX `idx_orders_status` (`status`),
    INDEX `idx_orders_idempotency_key` (`idempotency_key`),
    CONSTRAINT `fk_orders_user` FOREIGN KEY (`user_id`) REFERENCES `users`(`id`) ON DELETE CASCADE,
    CONSTRAINT `fk_orders_plan` FOREIGN KEY (`plan_id`) REFERENCES `plans`(`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS `payments` (
    `id` VARCHAR(128) PRIMARY KEY,
    `order_id` VARCHAR(128) NOT NULL,
    `user_id` VARCHAR(128) NOT NULL,
    `payment_provider` VARCHAR(64) NOT NULL DEFAULT 'shopier',
    `provider_payment_id` VARCHAR(128) NOT NULL DEFAULT '',
    `amount_cents` BIGINT NOT NULL,
    `currency` VARCHAR(3) NOT NULL DEFAULT 'TRY',
    `status` VARCHAR(32) NOT NULL,
    `idempotency_key` VARCHAR(128) UNIQUE,
    `error_message` TEXT NOT NULL,
    `metadata` TEXT NOT NULL,
    `paid_at` DATETIME(6) NULL,
    `tenant_id` VARCHAR(128) NOT NULL DEFAULT '',
    `created_at` DATETIME(6) NOT NULL,
    `updated_at` DATETIME(6) NOT NULL,
    INDEX `idx_payments_order_id` (`order_id`),
    INDEX `idx_payments_user_id` (`user_id`),
    INDEX `idx_payments_status` (`status`),
    INDEX `idx_payments_provider_id` (`provider_payment_id`),
    CONSTRAINT `fk_payments_order` FOREIGN KEY (`order_id`) REFERENCES `orders`(`id`) ON DELETE CASCADE,
    CONSTRAINT `fk_payments_user` FOREIGN KEY (`user_id`) REFERENCES `users`(`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS `subscriptions` (
    `id` VARCHAR(128) PRIMARY KEY,
    `user_id` VARCHAR(128) NOT NULL,
    `plan_id` VARCHAR(128) NOT NULL,
    `status` VARCHAR(32) NOT NULL,
    `current_period_start` DATETIME(6) NOT NULL,
    `current_period_end` DATETIME(6) NOT NULL,
    `cancel_at_period_end` TINYINT(1) NOT NULL DEFAULT 0,
    `cancelled_at` DATETIME(6) NULL,
    `tenant_id` VARCHAR(128) NOT NULL DEFAULT '',
    `created_at` DATETIME(6) NOT NULL,
    `updated_at` DATETIME(6) NOT NULL,
    INDEX `idx_subscriptions_user_id` (`user_id`),
    INDEX `idx_subscriptions_status` (`status`),
    INDEX `idx_subscriptions_period_end` (`current_period_end`),
    CONSTRAINT `fk_subscriptions_user` FOREIGN KEY (`user_id`) REFERENCES `users`(`id`) ON DELETE CASCADE,
    CONSTRAINT `fk_subscriptions_plan` FOREIGN KEY (`plan_id`) REFERENCES `plans`(`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS `payment_events` (
    `id` VARCHAR(128) PRIMARY KEY,
    `payment_id` VARCHAR(128) NULL,
    `event_type` VARCHAR(64) NOT NULL,
    `provider` VARCHAR(64) NOT NULL DEFAULT 'shopier',
    `provider_reference` VARCHAR(128) NOT NULL DEFAULT '',
    `signature` TEXT NOT NULL,
    `payload` LONGTEXT NOT NULL,
    `status` VARCHAR(32) NOT NULL DEFAULT 'processed',
    `tenant_id` VARCHAR(128) NOT NULL DEFAULT '',
    `processed_at` DATETIME(6) NOT NULL,
    `created_at` DATETIME(6) NOT NULL,
    UNIQUE KEY `uq_payment_events_ref` (`provider`, `provider_reference`, `event_type`),
    INDEX `idx_payment_events_ref` (`provider`, `provider_reference`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

INSERT IGNORE INTO `plans` (`id`, `name`, `code`, `description`, `price_cents`, `currency`, `billing_interval`, `interval_count`, `features`, `active`, `tenant_id`, `created_at`, `updated_at`)
VALUES
    ('plan_monthly', 'HydroCloud Pro (Aylık)', 'monthly', 'Tam donanımlı mağaza ve SaaS altyapısı - Aylık abonelik', 49900, 'TRY', 'month', 1, '["Sınırsız Ürün ve Sayfa", "Shopier Güvenli Ödeme", "Özel Alan Adı Bağlama", "Yapay Zeka Destekli Copilot", "7/24 Teknik Destek"]', 1, '', '2026-11-08 00:00:00', '2026-11-08 00:00:00'),
    ('plan_yearly', 'HydroCloud Pro (Yıllık)', 'yearly', 'Tam donanımlı mağaza ve SaaS altyapısı - Yıllık abonelik (2 Ay Ücretsiz)', 499000, 'TRY', 'year', 1, '["Sınırsız Ürün ve Sayfa", "Shopier Güvenli Ödeme", "Özel Alan Adı Bağlama", "Yapay Zeka Destekli Copilot", "Öncelikli 7/24 VIP Destek", "Yıllık Ödemede 2 Ay Hediye"]', 1, '', '2026-11-08 00:00:00', '2026-11-08 00:00:00');
