CREATE TABLE hydro_collections (
    tenant_id TEXT NOT NULL DEFAULT '',
    id TEXT NOT NULL,
    label TEXT NOT NULL,
    schema_json TEXT NOT NULL,
    version INTEGER NOT NULL DEFAULT 1,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    PRIMARY KEY (tenant_id, id)
);

CREATE TABLE hydro_collection_records (
    tenant_id TEXT NOT NULL DEFAULT '',
    collection_id TEXT NOT NULL,
    id TEXT NOT NULL,
    data_json TEXT NOT NULL,
    version INTEGER NOT NULL DEFAULT 1,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    PRIMARY KEY (tenant_id, collection_id, id),
    FOREIGN KEY (tenant_id, collection_id) REFERENCES hydro_collections(tenant_id, id) ON DELETE CASCADE
);
CREATE INDEX idx_hydro_collection_records_list ON hydro_collection_records(tenant_id, collection_id, updated_at DESC, id);

CREATE TABLE hydro_collection_unique (
    tenant_id TEXT NOT NULL DEFAULT '',
    collection_id TEXT NOT NULL,
    field_id TEXT NOT NULL,
    value_hash TEXT NOT NULL,
    record_id TEXT NOT NULL,
    PRIMARY KEY (tenant_id, collection_id, field_id, value_hash),
    FOREIGN KEY (tenant_id, collection_id, record_id) REFERENCES hydro_collection_records(tenant_id, collection_id, id) ON DELETE CASCADE
);

CREATE TABLE hydro_collection_relations (
    tenant_id TEXT NOT NULL DEFAULT '',
    source_collection_id TEXT NOT NULL,
    source_record_id TEXT NOT NULL,
    field_id TEXT NOT NULL,
    target_collection_id TEXT NOT NULL,
    target_record_id TEXT NOT NULL,
    PRIMARY KEY (tenant_id, source_collection_id, source_record_id, field_id, target_record_id),
    FOREIGN KEY (tenant_id, source_collection_id, source_record_id) REFERENCES hydro_collection_records(tenant_id, collection_id, id) ON DELETE CASCADE,
    FOREIGN KEY (tenant_id, target_collection_id, target_record_id) REFERENCES hydro_collection_records(tenant_id, collection_id, id) ON DELETE RESTRICT
);
CREATE INDEX idx_hydro_collection_relations_target ON hydro_collection_relations(tenant_id, target_collection_id, target_record_id);
