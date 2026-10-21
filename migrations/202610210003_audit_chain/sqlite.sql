-- Tamper evidence for the audit log. Every audit event can be chained: its
-- audit_chain row holds the event's place (seq) per tenant and a SHA-256 over
-- the previous hash and the event's fields. audit_chain_heads holds the latest
-- seq and hash per tenant; writers lock their row so the chain never forks.
-- Chain rows are immutable like the events they cover.
CREATE TABLE IF NOT EXISTS audit_chain (
    tenant_id TEXT NOT NULL DEFAULT '',
    seq INTEGER NOT NULL,
    event_id TEXT NOT NULL,
    prev_hash TEXT NOT NULL,
    chain_hash TEXT NOT NULL,
    PRIMARY KEY (tenant_id, seq)
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_audit_chain_event ON audit_chain(event_id);
CREATE TABLE IF NOT EXISTS audit_chain_heads (
    tenant_id TEXT PRIMARY KEY,
    seq INTEGER NOT NULL,
    last_hash TEXT NOT NULL
);
CREATE TRIGGER IF NOT EXISTS audit_chain_immutable_update
BEFORE UPDATE ON audit_chain
BEGIN
    SELECT RAISE(ABORT, 'audit chain rows are immutable');
END;
CREATE TRIGGER IF NOT EXISTS audit_chain_immutable_delete
BEFORE DELETE ON audit_chain
BEGIN
    SELECT RAISE(ABORT, 'audit chain rows are immutable');
END;
