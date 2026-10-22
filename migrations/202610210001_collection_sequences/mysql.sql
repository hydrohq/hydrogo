CREATE TABLE hydro_collection_sequences (
    tenant_id VARCHAR(128) NOT NULL DEFAULT '',
    collection_id VARCHAR(63) NOT NULL,
    field_id VARCHAR(63) NOT NULL,
    scope_key VARCHAR(190) NOT NULL,
    `last_value` BIGINT NOT NULL DEFAULT 0,
    PRIMARY KEY (tenant_id, collection_id, field_id, scope_key),
    CONSTRAINT hydro_collection_sequence_collection_fk FOREIGN KEY (tenant_id, collection_id) REFERENCES hydro_collections(tenant_id, id) ON DELETE CASCADE
);
