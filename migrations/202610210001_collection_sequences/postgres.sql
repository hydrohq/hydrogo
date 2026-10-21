CREATE TABLE hydro_collection_sequences (
    tenant_id TEXT NOT NULL DEFAULT '',
    collection_id TEXT NOT NULL,
    field_id TEXT NOT NULL,
    scope_key TEXT NOT NULL,
    last_value BIGINT NOT NULL DEFAULT 0,
    PRIMARY KEY (tenant_id, collection_id, field_id, scope_key),
    FOREIGN KEY (tenant_id, collection_id) REFERENCES hydro_collections(tenant_id, id) ON DELETE CASCADE
);
