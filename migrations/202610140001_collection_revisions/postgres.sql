CREATE TABLE hydro_collection_revisions (
    tenant_id TEXT NOT NULL DEFAULT '',
    collection_id TEXT NOT NULL,
    record_id TEXT NOT NULL,
    version INTEGER NOT NULL,
    data_json TEXT NOT NULL,
    actor TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL,
    PRIMARY KEY (tenant_id, collection_id, record_id, version),
    FOREIGN KEY (tenant_id, collection_id, record_id) REFERENCES hydro_collection_records(tenant_id, collection_id, id) ON DELETE CASCADE
);
