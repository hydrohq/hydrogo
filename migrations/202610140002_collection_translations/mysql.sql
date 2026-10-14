CREATE TABLE hydro_collection_translations (
    tenant_id VARCHAR(128) NOT NULL DEFAULT '',
    collection_id VARCHAR(63) NOT NULL,
    record_id VARCHAR(32) NOT NULL,
    locale VARCHAR(35) NOT NULL,
    data_json LONGTEXT NOT NULL,
    version INTEGER NOT NULL DEFAULT 1,
    updated_at VARCHAR(40) NOT NULL,
    PRIMARY KEY (tenant_id, collection_id, record_id, locale),
    CONSTRAINT hydro_collection_translation_record_fk FOREIGN KEY (tenant_id, collection_id, record_id) REFERENCES hydro_collection_records(tenant_id, collection_id, id) ON DELETE CASCADE
);
