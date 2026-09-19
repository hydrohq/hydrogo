ALTER TABLE hydro_forms ADD COLUMN tenant_id TEXT NOT NULL DEFAULT '';
CREATE INDEX idx_hydro_forms_tenant_id ON hydro_forms(tenant_id, id);
