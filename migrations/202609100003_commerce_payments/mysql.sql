CREATE TABLE commerce_payment_transactions (
    id VARCHAR(128) PRIMARY KEY,
    order_id VARCHAR(128) NOT NULL,
    provider VARCHAR(64) NOT NULL,
    provider_payment_id VARCHAR(191) NOT NULL,
    amount_cents BIGINT NOT NULL CHECK (amount_cents >= 0),
    currency CHAR(3) NOT NULL,
    status VARCHAR(24) NOT NULL,
    created_at DATETIME(6) NOT NULL,
    updated_at DATETIME(6) NOT NULL,
    UNIQUE (provider, provider_payment_id),
    CONSTRAINT commerce_payment_order_fk FOREIGN KEY (order_id) REFERENCES commerce_orders(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE commerce_payment_webhook_events (
    provider VARCHAR(64) NOT NULL,
    provider_event_id VARCHAR(191) NOT NULL,
    received_at DATETIME(6) NOT NULL,
    PRIMARY KEY (provider, provider_event_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
