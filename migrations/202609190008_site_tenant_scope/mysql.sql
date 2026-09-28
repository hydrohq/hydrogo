ALTER TABLE site_settings ADD COLUMN tenant_id VARCHAR(128) NOT NULL DEFAULT '';
ALTER TABLE site_settings DROP PRIMARY KEY, ADD PRIMARY KEY (tenant_id, `key`);

ALTER TABLE navigation_items ADD COLUMN tenant_id VARCHAR(128) NOT NULL DEFAULT '';
ALTER TABLE navigation_items DROP INDEX idx_navigation_items_locale_position;
ALTER TABLE navigation_items DROP INDEX locale;
ALTER TABLE navigation_items DROP PRIMARY KEY, ADD PRIMARY KEY (tenant_id, id);
CREATE UNIQUE INDEX idx_navigation_items_tenant_locale_position ON navigation_items(tenant_id, locale, position);

ALTER TABLE page_redirects ADD COLUMN tenant_id VARCHAR(128) NOT NULL DEFAULT '';
ALTER TABLE page_redirects DROP PRIMARY KEY, ADD PRIMARY KEY (tenant_id, source_path);

ALTER TABLE form_inbox ADD COLUMN tenant_id VARCHAR(128) NOT NULL DEFAULT '';
CREATE INDEX idx_form_inbox_tenant_key_created ON form_inbox(tenant_id, form_key, created_at);
