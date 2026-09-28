CREATE TABLE hydro_collections (
    tenant_id VARCHAR(128) NOT NULL DEFAULT '',
    id VARCHAR(63) NOT NULL,
    label VARCHAR(128) NOT NULL,
    schema_json LONGTEXT NOT NULL,
    version INTEGER NOT NULL DEFAULT 1,
    created_at VARCHAR(40) NOT NULL,
    updated_at VARCHAR(40) NOT NULL,
    PRIMARY KEY (tenant_id, id)
);

CREATE TABLE hydro_collection_records (
    tenant_id VARCHAR(128) NOT NULL DEFAULT '',
    collection_id VARCHAR(63) NOT NULL,
    id VARCHAR(32) NOT NULL,
    data_json LONGTEXT NOT NULL,
    version INTEGER NOT NULL DEFAULT 1,
    created_at VARCHAR(40) NOT NULL,
    updated_at VARCHAR(40) NOT NULL,
    PRIMARY KEY (tenant_id, collection_id, id),
    CONSTRAINT hydro_collection_record_schema_fk FOREIGN KEY (tenant_id, collection_id) REFERENCES hydro_collections(tenant_id, id) ON DELETE CASCADE
);
CREATE INDEX idx_hydro_collection_records_list ON hydro_collection_records(tenant_id, collection_id, updated_at, id);

CREATE TABLE hydro_collection_unique (
    tenant_id VARCHAR(128) NOT NULL DEFAULT '',
    collection_id VARCHAR(63) NOT NULL,
    field_id VARCHAR(63) NOT NULL,
    value_hash CHAR(64) NOT NULL,
    record_id VARCHAR(32) NOT NULL,
    PRIMARY KEY (tenant_id, collection_id, field_id, value_hash),
    CONSTRAINT hydro_collection_unique_record_fk FOREIGN KEY (tenant_id, collection_id, record_id) REFERENCES hydro_collection_records(tenant_id, collection_id, id) ON DELETE CASCADE
);

CREATE TABLE hydro_collection_relations (
    tenant_id VARCHAR(128) NOT NULL DEFAULT '',
    source_collection_id VARCHAR(63) NOT NULL,
    source_record_id VARCHAR(32) NOT NULL,
    field_id VARCHAR(63) NOT NULL,
    target_collection_id VARCHAR(63) NOT NULL,
    target_record_id VARCHAR(32) NOT NULL,
    PRIMARY KEY (tenant_id, source_collection_id, source_record_id, field_id, target_record_id),
    CONSTRAINT hydro_collection_relation_source_fk FOREIGN KEY (tenant_id, source_collection_id, source_record_id) REFERENCES hydro_collection_records(tenant_id, collection_id, id) ON DELETE CASCADE,
    CONSTRAINT hydro_collection_relation_target_fk FOREIGN KEY (tenant_id, target_collection_id, target_record_id) REFERENCES hydro_collection_records(tenant_id, collection_id, id) ON DELETE RESTRICT
);
CREATE INDEX idx_hydro_collection_relations_target ON hydro_collection_relations(tenant_id, target_collection_id, target_record_id);
