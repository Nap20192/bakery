-- name: CreatePasswordAuthUser :one
INSERT INTO auth_users (
    department_id,
    username,
    password_hash,
    metadata_json,
    role,
    created_at,
    updated_at
) VALUES (
    sqlc.narg(department_id),
    sqlc.arg(username),
    sqlc.arg(password_hash),
    sqlc.arg(metadata_json),
    sqlc.arg(role),
    sqlc.arg(created_at),
    sqlc.arg(updated_at)
)
ON CONFLICT(username) WHERE username <> '' DO UPDATE SET
    department_id = excluded.department_id,
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

-- name: ListAuthUsersByDepartmentID :many
SELECT *
FROM auth_users
WHERE department_id = sqlc.arg(department_id)
  AND telegram_id IS NOT NULL
ORDER BY id;

-- name: LinkTelegramAuthUser :one
UPDATE auth_users
SET
    telegram_id = sqlc.arg(telegram_id),
    telegram_username = sqlc.narg(telegram_username),
    updated_at = sqlc.arg(updated_at)
WHERE id = sqlc.arg(id)
RETURNING *;

-- name: UnlinkTelegramAuthUser :exec
UPDATE auth_users
SET
    telegram_id = NULL,
    updated_at = sqlc.arg(updated_at)
WHERE telegram_id = sqlc.arg(telegram_id);

-- name: UpsertTelegramAuthUserDepartment :one
INSERT INTO auth_users (
    telegram_id,
    telegram_username,
    department_id,
    role,
    created_at,
    updated_at
) VALUES (
    sqlc.arg(telegram_id),
    sqlc.narg(telegram_username),
    sqlc.arg(department_id),
    'user',
    sqlc.arg(created_at),
    sqlc.arg(updated_at)
)
ON CONFLICT(telegram_id) DO UPDATE SET
    telegram_username = excluded.telegram_username,
    department_id = excluded.department_id,
    updated_at = excluded.updated_at
RETURNING *;
