-- A TOTP code stays valid for its whole 30-second step (and one step of
-- clock skew either side). Remembering the last step accepted per user makes
-- an observed or phished code single-use instead of replayable in that window.
ALTER TABLE user_mfa ADD COLUMN last_totp_step INTEGER NOT NULL DEFAULT 0;
