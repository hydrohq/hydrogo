UPDATE admin_notifications SET tenant_id = '' WHERE tenant_id IS NULL;
ALTER TABLE admin_notifications MODIFY tenant_id VARCHAR(128) NOT NULL DEFAULT '';
