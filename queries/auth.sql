-- name: CreateAuthUser :one
INSERT INTO auth_users (
    telegram_id,
    username,
    role,
    created_at,
    updated_at
) VALUES (
    sqlc.arg(telegram_id),
    sqlc.arg(username),
    sqlc.arg(role),
    sqlc.arg(created_at),
    sqlc.arg(updated_at)
)
ON CONFLICT(telegram_id) DO UPDATE SET
    username = excluded.username,
    role = excluded.role,
    updated_at = excluded.updated_at
RETURNING *;

-- name: GetAuthUserByTelegramID :one
SELECT *
FROM auth_users
WHERE telegram_id = sqlc.arg(telegram_id);
