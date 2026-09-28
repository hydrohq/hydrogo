CREATE TABLE sectoral_team (
    id VARCHAR(128) NOT NULL,
    tenant_id VARCHAR(128) NOT NULL DEFAULT '',
    locale VARCHAR(16) NOT NULL,
    slug VARCHAR(256) NOT NULL,
    title VARCHAR(256) NOT NULL,
    status VARCHAR(16) NOT NULL DEFAULT 'draft' CHECK (status IN ('draft','published','archived')),
    featured TINYINT NOT NULL DEFAULT 0 CHECK (featured IN (0,1)),
    sort_order INT NOT NULL DEFAULT 0 CHECK (sort_order BETWEEN -100000 AND 100000),
    version INT NOT NULL DEFAULT 1 CHECK (version > 0),
    payload LONGTEXT NOT NULL CHECK (CHAR_LENGTH(payload) <= 1048576),
    updated_at DATETIME(6) NOT NULL,
    PRIMARY KEY (id),
    UNIQUE KEY sectoral_team_tenant_locale_slug_uq (tenant_id,locale,slug),
    KEY sectoral_team_public_order_idx (tenant_id,status,locale,featured,sort_order,updated_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
