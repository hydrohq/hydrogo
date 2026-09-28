CREATE TABLE site_settings (
    `key` VARCHAR(191) PRIMARY KEY,
    `value` LONGTEXT NOT NULL,
    is_public TINYINT(1) NOT NULL DEFAULT 0,
    updated_at DATETIME(6) NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE navigation_items (
    id VARCHAR(128) PRIMARY KEY,
    locale VARCHAR(16) NOT NULL,
    label VARCHAR(512) NOT NULL,
    href VARCHAR(2048) NOT NULL,
    position INTEGER NOT NULL,
    is_visible TINYINT(1) NOT NULL DEFAULT 1,
    updated_at DATETIME(6) NOT NULL,
    UNIQUE (locale, position, id),
    INDEX idx_navigation_items_locale (locale, position)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE page_redirects (
    source_path VARCHAR(512) PRIMARY KEY,
    target_path VARCHAR(2048) NOT NULL,
    status INTEGER NOT NULL CHECK (status IN (301, 308)),
    created_at DATETIME(6) NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE form_inbox (
    id VARCHAR(128) PRIMARY KEY,
    form_key VARCHAR(128) NOT NULL,
    email VARCHAR(320) NOT NULL,
    payload LONGTEXT NOT NULL,
    source_ip_hash CHAR(64) NOT NULL,
    status VARCHAR(32) NOT NULL DEFAULT 'new',
    created_at DATETIME(6) NOT NULL,
    INDEX idx_form_inbox_key_created (form_key, created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
