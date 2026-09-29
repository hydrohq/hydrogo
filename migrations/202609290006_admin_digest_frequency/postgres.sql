ALTER TABLE admin_digest_preferences ADD COLUMN frequency TEXT NOT NULL DEFAULT 'weekly' CHECK (frequency IN ('weekly','monthly'));
