CREATE TABLE security_ip_bans (
    address TEXT PRIMARY KEY,
    expires_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX idx_security_ip_bans_expires ON security_ip_bans(expires_at);
