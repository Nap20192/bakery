-- +goose Up
-- telegram_username is the login/identity for the bot and mini app, so it must
-- be unique (telegram_id and username are already unique). Replace the plain
-- index with a unique partial index over non-empty values.
DROP INDEX IF EXISTS idx_auth_users_telegram_username;

CREATE UNIQUE INDEX IF NOT EXISTS idx_auth_users_telegram_username
ON auth_users(telegram_username)
WHERE telegram_username IS NOT NULL AND telegram_username <> '';

-- +goose Down
DROP INDEX IF EXISTS idx_auth_users_telegram_username;

CREATE INDEX IF NOT EXISTS idx_auth_users_telegram_username
ON auth_users(telegram_username)
WHERE telegram_username IS NOT NULL AND telegram_username <> '';
