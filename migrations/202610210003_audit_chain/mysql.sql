-- Tamper evidence for the audit log. Every audit event can be chained: its
-- audit_chain row holds the event's place (seq) per tenant and a SHA-256 over
-- the previous hash and the event's fields. audit_chain_heads holds the latest
-- seq and hash per tenant; writers lock their row so the chain never forks.
-- Chain rows are immutable like the events they cover.
CREATE TABLE IF NOT EXISTS audit_chain (
    tenant_id VARCHAR(128) NOT NULL DEFAULT '',
    seq BIGINT NOT NULL,
    event_id VARCHAR(128) NOT NULL,
    prev_hash VARCHAR(64) NOT NULL,
    chain_hash VARCHAR(64) NOT NULL,
    PRIMARY KEY (tenant_id, seq),
    UNIQUE KEY idx_audit_chain_event (event_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
CREATE TABLE IF NOT EXISTS audit_chain_heads (
    tenant_id VARCHAR(128) NOT NULL PRIMARY KEY,
    seq BIGINT NOT NULL,
    last_hash VARCHAR(64) NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

DROP TRIGGER IF EXISTS audit_chain_immutable_update;
DROP TRIGGER IF EXISTS audit_chain_immutable_delete;
CREATE TRIGGER audit_chain_immutable_update BEFORE UPDATE ON audit_chain FOR EACH ROW
BEGIN
    SIGNAL SQLSTATE '45000' SET MESSAGE_TEXT = 'audit chain rows are immutable';
END;
CREATE TRIGGER audit_chain_immutable_delete BEFORE DELETE ON audit_chain FOR EACH ROW
BEGIN
    SIGNAL SQLSTATE '45000' SET MESSAGE_TEXT = 'audit chain rows are immutable';
END;
