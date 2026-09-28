CREATE TABLE push_subscriptions (
  tenant_id VARCHAR(191) NOT NULL DEFAULT '',
  id CHAR(64) NOT NULL,
  user_id VARCHAR(191) NOT NULL,
  platform VARCHAR(16) NOT NULL,
  endpoint TEXT NOT NULL,
  keys_json LONGTEXT NOT NULL,
  active TINYINT NOT NULL DEFAULT 1,
  created_at DATETIME(6) NOT NULL,
  updated_at DATETIME(6) NOT NULL,
  PRIMARY KEY (tenant_id,id),
  KEY idx_push_subscriptions_broadcast (tenant_id,platform,active,created_at)
);
