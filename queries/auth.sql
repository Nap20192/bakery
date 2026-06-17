-- name: CreatePasswordAuthUser :one
INSERT INTO auth_users (
    department_id,
    username,
    telegram_username,
    password_hash,
    metadata_json,
    role,
    created_at,
    updated_at
) VALUES (
    sqlc.narg(department_id),
    sqlc.arg(username),
    sqlc.narg(telegram_username),
    sqlc.arg(password_hash),
    sqlc.arg(metadata_json),
    sqlc.arg(role),
    sqlc.arg(created_at),
    sqlc.arg(updated_at)
)
ON CONFLICT(username) WHERE username <> '' DO UPDATE SET
    department_id = excluded.department_id,
    telegram_username = excluded.telegram_username,
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

-- name: GetAuthUserByTelegramUsername :one
SELECT *
FROM auth_users
WHERE telegram_username = sqlc.arg(telegram_username)::text
  AND telegram_id IS NOT NULL
ORDER BY id
LIMIT 1;

-- name: ListAuthUsersByRole :many
SELECT *
FROM auth_users
WHERE role = sqlc.arg(role)
  AND telegram_id IS NOT NULL
ORDER BY id;

-- name: GetAuthUserByID :one
SELECT *
FROM auth_users
WHERE id = sqlc.arg(id);

-- name: ListAuthUsers :many
SELECT *
FROM auth_users
ORDER BY id;

-- name: UpdateAuthUserRole :one
UPDATE auth_users
SET role = sqlc.arg(role),
    updated_at = sqlc.arg(updated_at)
WHERE id = sqlc.arg(id)
RETURNING *;

-- name: BindTelegramID :one
UPDATE auth_users
SET telegram_id = sqlc.arg(telegram_id),
    updated_at = sqlc.arg(updated_at)
WHERE id = sqlc.arg(id)
RETURNING *;

-- name: UpdateAuthUserPassword :one
UPDATE auth_users
SET password_hash = sqlc.arg(password_hash),
    updated_at = sqlc.arg(updated_at)
WHERE id = sqlc.arg(id)
RETURNING *;

-- name: DeleteAuthUser :exec
DELETE FROM auth_users WHERE id = sqlc.arg(id);
