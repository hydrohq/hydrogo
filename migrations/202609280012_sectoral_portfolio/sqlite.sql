CREATE TABLE sectoral_portfolio (
    id TEXT NOT NULL CHECK (length(id) BETWEEN 1 AND 128),
    tenant_id TEXT NOT NULL DEFAULT '' CHECK (length(tenant_id) <= 128),
    locale TEXT NOT NULL CHECK (length(locale) BETWEEN 1 AND 16),
    slug TEXT NOT NULL CHECK (length(slug) BETWEEN 1 AND 256),
    title TEXT NOT NULL CHECK (length(title) BETWEEN 1 AND 256),
    status TEXT NOT NULL DEFAULT 'draft' CHECK (status IN ('draft','published','archived')),
    featured INTEGER NOT NULL DEFAULT 0 CHECK (featured IN (0,1)),
    sort_order INTEGER NOT NULL DEFAULT 0 CHECK (sort_order BETWEEN -100000 AND 100000),
    version INTEGER NOT NULL DEFAULT 1 CHECK (version > 0),
    payload TEXT NOT NULL CHECK (length(payload) <= 1048576),
    updated_at TEXT NOT NULL,
    PRIMARY KEY (id),
    UNIQUE (tenant_id,locale,slug)
);
CREATE INDEX sectoral_portfolio_public_order_idx ON sectoral_portfolio(tenant_id,status,locale,featured,sort_order,updated_at);
