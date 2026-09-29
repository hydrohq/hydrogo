CREATE TABLE mail_branding_settings (
    singleton_id INT PRIMARY KEY CHECK (singleton_id = 1),
    legal_name VARCHAR(256) NOT NULL DEFAULT '',
    physical_address VARCHAR(512) NOT NULL DEFAULT '',
    privacy_policy_url VARCHAR(2048) NOT NULL DEFAULT '',
    footer_notice VARCHAR(1024) NOT NULL DEFAULT '',
    logo_asset_id VARCHAR(128) NOT NULL DEFAULT '',
    primary_color VARCHAR(7) NOT NULL DEFAULT '#2563eb',
    email_footer_text VARCHAR(1024) NOT NULL DEFAULT '',
    updated_at DATETIME(6) NOT NULL
);
