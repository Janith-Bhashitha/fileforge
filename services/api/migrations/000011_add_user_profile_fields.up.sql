-- Profile editing (display_name was write-once at registration with no way
-- to change it - the "name not saving" defect), an avatar image key, and
-- password-reset tokens.
ALTER TABLE users ADD COLUMN avatar_key TEXT;
ALTER TABLE users ADD COLUMN reset_token TEXT;
ALTER TABLE users ADD COLUMN reset_token_expires_at TIMESTAMPTZ;

-- Looked up by token on the reset-password step; NULL for every user who
-- has never requested a reset, so a plain index (not unique) is correct -
-- uniqueness is enforced at generation time instead, by using a random UUID.
CREATE INDEX idx_users_reset_token ON users (reset_token) WHERE reset_token IS NOT NULL;
