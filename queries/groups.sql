-- name: UpsertGroup :one
INSERT INTO groups (
    code,
    name
) VALUES (
    sqlc.arg(code),
    sqlc.arg(name)
)
ON CONFLICT(code) DO UPDATE SET
    name = excluded.name
RETURNING *;

-- name: GetGroupByCode :one
SELECT *
FROM groups
WHERE code = sqlc.arg(code);

-- name: ListGroups :many
SELECT *
FROM groups
ORDER BY name, code;
