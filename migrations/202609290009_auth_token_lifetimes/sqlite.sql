CREATE TABLE auth_token_lifetimes (
    singleton_id INTEGER PRIMARY KEY CHECK (singleton_id = 1),
    password_reset_minutes INTEGER NOT NULL CHECK (password_reset_minutes IN (15, 60, 1440)),
    invitation_hours INTEGER NOT NULL CHECK (invitation_hours IN (24, 72, 168)),
    updated_at TEXT NOT NULL
);
