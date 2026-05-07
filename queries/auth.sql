-- name: CreatePasswordAuthUser :one
INSERT INTO auth_users (
    username,
    password_hash,
    metadata_json,
    role,
    created_at,
    updated_at
) VALUES (
    sqlc.arg(username),
    sqlc.arg(password_hash),
    sqlc.arg(metadata_json),
    sqlc.arg(role),
    sqlc.arg(created_at),
    sqlc.arg(updated_at)
)
ON CONFLICT(username) WHERE username <> '' DO UPDATE SET
    password_hash = excluded.password_hash,
    metadata_json = excluded.metadata_json,
    role = excluded.role,
    updated_at = excluded.updated_at
RETURNING *;

-- name: GetAuthUserByTelegramID :one
SELECT *
FROM auth_users
WHERE telegram_id = sqlc.arg(telegram_id);

-- name: GetAuthUserByUsername :one
SELECT *
FROM auth_users
WHERE username = sqlc.arg(username);

-- name: LinkTelegramAuthUser :one
UPDATE auth_users
SET
    telegram_id = sqlc.arg(telegram_id),
    updated_at = sqlc.arg(updated_at)
WHERE id = sqlc.arg(id)
RETURNING *;

-- name: UnlinkTelegramAuthUser :exec
UPDATE auth_users
SET
    telegram_id = NULL,
    updated_at = sqlc.arg(updated_at)
WHERE telegram_id = sqlc.arg(telegram_id);
