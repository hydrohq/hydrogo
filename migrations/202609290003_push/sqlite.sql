CREATE TABLE push_subscriptions (
  tenant_id TEXT NOT NULL DEFAULT '',
  id TEXT NOT NULL,
  user_id TEXT NOT NULL,
  platform TEXT NOT NULL CHECK (platform IN ('web','mobile')),
  endpoint TEXT NOT NULL,
  keys_json TEXT NOT NULL DEFAULT '{}',
  active INTEGER NOT NULL DEFAULT 1 CHECK (active IN (0,1)),
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  PRIMARY KEY (tenant_id,id),
  UNIQUE (tenant_id,user_id,platform,endpoint)
);
CREATE INDEX idx_push_subscriptions_broadcast ON push_subscriptions(tenant_id,platform,active,created_at);
