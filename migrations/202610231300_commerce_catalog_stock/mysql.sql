-- Stock lives here, not on a content-type record: selling the last unit
-- needs an atomic "decrement if enough is available" update, which the
-- generic collections store (optimistic version-locking only) can't give.
-- variant_id is a content-type record id (content-types/commerce/variants.yaml),
-- not a foreign key: content records live in the generic
-- hydro_collection_records table, which can't be referenced across dialects.
CREATE TABLE IF NOT EXISTS commerce_stock (
    id VARCHAR(128) PRIMARY KEY,
    variant_id VARCHAR(128) NOT NULL,
    available INTEGER NOT NULL DEFAULT 0 CHECK (available >= 0),
    reserved INTEGER NOT NULL DEFAULT 0 CHECK (reserved >= 0),
    created_at DATETIME(6) NOT NULL,
    updated_at DATETIME(6) NOT NULL,
    tenant_id VARCHAR(128) NOT NULL DEFAULT '',
    UNIQUE KEY idx_commerce_stock_variant (tenant_id, variant_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
