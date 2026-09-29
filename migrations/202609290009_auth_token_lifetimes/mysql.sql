CREATE TABLE auth_token_lifetimes (
    singleton_id INT PRIMARY KEY CHECK (singleton_id = 1),
    password_reset_minutes INT NOT NULL CHECK (password_reset_minutes IN (15, 60, 1440)),
    invitation_hours INT NOT NULL CHECK (invitation_hours IN (24, 72, 168)),
    updated_at DATETIME(6) NOT NULL
);
