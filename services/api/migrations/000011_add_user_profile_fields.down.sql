DROP INDEX idx_users_reset_token;
ALTER TABLE users DROP COLUMN reset_token_expires_at;
ALTER TABLE users DROP COLUMN reset_token;
ALTER TABLE users DROP COLUMN avatar_key;
