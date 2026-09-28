CREATE TABLE sectoral_events (
    id TEXT PRIMARY KEY CHECK (length(id) BETWEEN 1 AND 128),
    tenant_id TEXT NOT NULL DEFAULT '' CHECK (length(tenant_id) <= 128),
    locale TEXT NOT NULL CHECK (length(locale) BETWEEN 1 AND 16),
    slug TEXT NOT NULL CHECK (length(slug) BETWEEN 1 AND 256),
    title TEXT NOT NULL CHECK (length(title) BETWEEN 1 AND 256),
    status TEXT NOT NULL DEFAULT 'draft' CHECK (status IN ('draft','published','archived')),
    featured BOOLEAN NOT NULL DEFAULT FALSE,
    sort_order INTEGER NOT NULL DEFAULT 0 CHECK (sort_order BETWEEN -100000 AND 100000),
    version INTEGER NOT NULL DEFAULT 1 CHECK (version > 0),
    payload TEXT NOT NULL CHECK (length(payload) <= 1048576),
    updated_at TIMESTAMPTZ NOT NULL,
    UNIQUE (tenant_id,locale,slug)
);
CREATE INDEX sectoral_events_public_order_idx ON sectoral_events(tenant_id,status,locale,featured,sort_order,updated_at);

ALTER TABLE sectoral_events ENABLE ROW LEVEL SECURITY;
ALTER TABLE sectoral_events FORCE ROW LEVEL SECURITY;
CREATE POLICY hydrogo_tenant_isolation ON sectoral_events
USING ((coalesce(nullif(current_setting('app.tenant_rls_enforced', true), ''), 'false') <> 'true') OR tenant_id = nullif(current_setting('app.current_tenant_id', true), ''))
WITH CHECK ((coalesce(nullif(current_setting('app.tenant_rls_enforced', true), ''), 'false') <> 'true') OR tenant_id = nullif(current_setting('app.current_tenant_id', true), ''));
