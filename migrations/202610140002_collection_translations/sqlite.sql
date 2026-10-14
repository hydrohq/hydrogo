CREATE TABLE hydro_collection_translations (
    tenant_id TEXT NOT NULL DEFAULT '',
    collection_id TEXT NOT NULL,
    record_id TEXT NOT NULL,
    locale TEXT NOT NULL,
    data_json TEXT NOT NULL,
    version INTEGER NOT NULL DEFAULT 1,
    updated_at TEXT NOT NULL,
    PRIMARY KEY (tenant_id, collection_id, record_id, locale),
    FOREIGN KEY (tenant_id, collection_id, record_id) REFERENCES hydro_collection_records(tenant_id, collection_id, id) ON DELETE CASCADE
);
