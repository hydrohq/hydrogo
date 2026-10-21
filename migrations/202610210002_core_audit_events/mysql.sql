-- The audit log belongs to the core: projects without the pages module (the
-- crm preset) get the same table the pages migrations build, in its final
-- shape. Where the pages migrations already created it, the table and indexes
-- stay as they are and the triggers are recreated identically.
CREATE TABLE IF NOT EXISTS audit_events (
    id VARCHAR(128) PRIMARY KEY,
    action VARCHAR(128) NOT NULL,
    target_id VARCHAR(128) NOT NULL,
    request_id VARCHAR(128),
    tenant_id VARCHAR(128) NOT NULL DEFAULT '',
    actor_id VARCHAR(128) NOT NULL DEFAULT '',
    resource_type VARCHAR(128) NOT NULL DEFAULT '',
    created_at VARCHAR(40) NOT NULL DEFAULT '1970-01-01T00:00:00.000000000Z',
    before_json VARCHAR(4096) NOT NULL DEFAULT '{}',
    after_json VARCHAR(4096) NOT NULL DEFAULT '{}',
    INDEX idx_audit_tenant_target (tenant_id, target_id, id),
    INDEX idx_audit_events_tenant_created (tenant_id, created_at, id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

DROP TRIGGER IF EXISTS audit_events_immutable_update;
DROP TRIGGER IF EXISTS audit_events_immutable_delete;
CREATE TRIGGER audit_events_immutable_update BEFORE UPDATE ON audit_events FOR EACH ROW
BEGIN
    SIGNAL SQLSTATE '45000' SET MESSAGE_TEXT = 'audit events are immutable';
END;
CREATE TRIGGER audit_events_immutable_delete BEFORE DELETE ON audit_events FOR EACH ROW
BEGIN
    SIGNAL SQLSTATE '45000' SET MESSAGE_TEXT = 'audit events are immutable';
END;
