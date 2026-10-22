-- Tamper evidence for the audit log. Every audit event can be chained: its
-- audit_chain row holds the event's place (seq) per tenant and a SHA-256 over
-- the previous hash and the event's fields. audit_chain_heads holds the latest
-- seq and hash per tenant; writers lock their row so the chain never forks.
-- Chain rows are immutable like the events they cover.
CREATE TABLE IF NOT EXISTS audit_chain (
    tenant_id TEXT NOT NULL DEFAULT '',
    seq BIGINT NOT NULL,
    event_id TEXT NOT NULL,
    prev_hash TEXT NOT NULL,
    chain_hash TEXT NOT NULL,
    PRIMARY KEY (tenant_id, seq)
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_audit_chain_event ON audit_chain(event_id);
CREATE TABLE IF NOT EXISTS audit_chain_heads (
    tenant_id TEXT PRIMARY KEY,
    seq BIGINT NOT NULL,
    last_hash TEXT NOT NULL
);

CREATE OR REPLACE FUNCTION hydro_reject_audit_chain_mutation() RETURNS trigger AS $$
BEGIN
    RAISE EXCEPTION 'audit chain rows are immutable';
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS audit_chain_immutable ON audit_chain;
CREATE TRIGGER audit_chain_immutable
BEFORE UPDATE OR DELETE ON audit_chain
FOR EACH ROW EXECUTE FUNCTION hydro_reject_audit_chain_mutation();

ALTER TABLE audit_chain ENABLE ROW LEVEL SECURITY;
ALTER TABLE audit_chain FORCE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS hydrogo_tenant_isolation ON audit_chain;
CREATE POLICY hydrogo_tenant_isolation ON audit_chain
    USING ((coalesce(nullif(current_setting('app.tenant_rls_enforced', true), ''), 'false') <> 'true') OR tenant_id = nullif(current_setting('app.current_tenant_id', true), ''))
    WITH CHECK ((coalesce(nullif(current_setting('app.tenant_rls_enforced', true), ''), 'false') <> 'true') OR tenant_id = nullif(current_setting('app.current_tenant_id', true), ''));

ALTER TABLE audit_chain_heads ENABLE ROW LEVEL SECURITY;
ALTER TABLE audit_chain_heads FORCE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS hydrogo_tenant_isolation ON audit_chain_heads;
CREATE POLICY hydrogo_tenant_isolation ON audit_chain_heads
    USING ((coalesce(nullif(current_setting('app.tenant_rls_enforced', true), ''), 'false') <> 'true') OR tenant_id = nullif(current_setting('app.current_tenant_id', true), ''))
    WITH CHECK ((coalesce(nullif(current_setting('app.tenant_rls_enforced', true), ''), 'false') <> 'true') OR tenant_id = nullif(current_setting('app.current_tenant_id', true), ''));
