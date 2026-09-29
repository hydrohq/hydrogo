CREATE TABLE mail_branding_settings (
    singleton_id INTEGER PRIMARY KEY CHECK (singleton_id = 1),
    legal_name TEXT NOT NULL DEFAULT '',
    physical_address TEXT NOT NULL DEFAULT '',
    privacy_policy_url TEXT NOT NULL DEFAULT '',
    footer_notice TEXT NOT NULL DEFAULT '',
    logo_asset_id TEXT NOT NULL DEFAULT '',
    primary_color TEXT NOT NULL DEFAULT '#2563eb',
    email_footer_text TEXT NOT NULL DEFAULT '',
    updated_at TEXT NOT NULL
);
