CREATE TABLE site_settings_tenant (
    tenant_id TEXT NOT NULL DEFAULT '',
    key TEXT NOT NULL,
    value TEXT NOT NULL,
    is_public INTEGER NOT NULL DEFAULT 0,
    updated_at TEXT NOT NULL,
    PRIMARY KEY (tenant_id, key)
);
INSERT INTO site_settings_tenant (tenant_id,key,value,is_public,updated_at) SELECT '',key,value,is_public,updated_at FROM site_settings;
DROP TABLE site_settings;
ALTER TABLE site_settings_tenant RENAME TO site_settings;

CREATE TABLE navigation_items_tenant (
    tenant_id TEXT NOT NULL DEFAULT '',
    id TEXT NOT NULL,
    locale TEXT NOT NULL,
    label TEXT NOT NULL,
    href TEXT NOT NULL,
    position INTEGER NOT NULL,
    is_visible INTEGER NOT NULL DEFAULT 1,
    updated_at TEXT NOT NULL,
    PRIMARY KEY (tenant_id, id),
    UNIQUE (tenant_id, locale, position)
);
INSERT INTO navigation_items_tenant (tenant_id,id,locale,label,href,position,is_visible,updated_at) SELECT '',id,locale,label,href,position,is_visible,updated_at FROM navigation_items;
DROP TABLE navigation_items;
ALTER TABLE navigation_items_tenant RENAME TO navigation_items;
CREATE UNIQUE INDEX idx_navigation_items_tenant_locale_position ON navigation_items(tenant_id, locale, position);

CREATE TABLE page_redirects_tenant (
    tenant_id TEXT NOT NULL DEFAULT '',
    source_path TEXT NOT NULL,
    target_path TEXT NOT NULL,
    status INTEGER NOT NULL CHECK (status IN (301, 308)),
    created_at TEXT NOT NULL,
    PRIMARY KEY (tenant_id, source_path)
);
INSERT INTO page_redirects_tenant (tenant_id,source_path,target_path,status,created_at) SELECT '',source_path,target_path,status,created_at FROM page_redirects;
DROP TABLE page_redirects;
ALTER TABLE page_redirects_tenant RENAME TO page_redirects;

ALTER TABLE form_inbox ADD COLUMN tenant_id TEXT NOT NULL DEFAULT '';
CREATE INDEX idx_form_inbox_tenant_key_created ON form_inbox(tenant_id, form_key, created_at);
