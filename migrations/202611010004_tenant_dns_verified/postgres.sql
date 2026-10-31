ALTER TABLE tenant_governance ADD COLUMN IF NOT EXISTS dns_verified BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE tenant_governance DROP CONSTRAINT IF EXISTS tenant_governance_lifecycle_state_check;
ALTER TABLE tenant_governance ADD CONSTRAINT tenant_governance_lifecycle_state_check CHECK (lifecycle_state IN ('provisioning', 'active', 'readonly', 'suspended', 'archived', 'demo'));
