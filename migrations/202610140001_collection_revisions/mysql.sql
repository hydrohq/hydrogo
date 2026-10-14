CREATE TABLE hydro_collection_revisions (
    tenant_id VARCHAR(128) NOT NULL DEFAULT '',
    collection_id VARCHAR(63) NOT NULL,
    record_id VARCHAR(32) NOT NULL,
    version INTEGER NOT NULL,
    data_json LONGTEXT NOT NULL,
    actor VARCHAR(320) NOT NULL DEFAULT '',
    created_at VARCHAR(40) NOT NULL,
    PRIMARY KEY (tenant_id, collection_id, record_id, version),
    CONSTRAINT hydro_collection_revision_record_fk FOREIGN KEY (tenant_id, collection_id, record_id) REFERENCES hydro_collection_records(tenant_id, collection_id, id) ON DELETE CASCADE
);
