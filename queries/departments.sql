-- name: GetDepartmentByID :one
SELECT *
FROM departments
WHERE id = sqlc.arg(id);

-- name: GetDepartmentByCode :one
SELECT *
FROM departments
WHERE code = sqlc.arg(code);

-- name: ListDepartments :many
SELECT *
FROM departments
WHERE type = COALESCE(sqlc.narg('type'), type)
ORDER BY type, name, id;

-- name: AssignUserDepartment :one
UPDATE auth_users
SET
    department_id = sqlc.narg(department_id),
    updated_at = sqlc.arg(updated_at)
WHERE id = sqlc.arg(user_id)
RETURNING *;
