-- name: KitchenInsert :one
INSERT INTO kitchens (name, address, iiko_id)
VALUES ($1, $2, $3)
RETURNING *;

-- name: KitchenGetByID :one
SELECT * FROM kitchens
WHERE id = $1 AND deleted_at IS NULL;

-- name: KitchenGetByIikoID :one
SELECT * FROM kitchens
WHERE iiko_id = $1 AND deleted_at IS NULL;

-- name: KitchenList :many
SELECT * FROM kitchens
WHERE deleted_at IS NULL
ORDER BY name;

-- name: KitchenUpdate :one
UPDATE kitchens
SET name = $2, address = $3, iiko_id = $4
WHERE id = $1 AND deleted_at IS NULL
RETURNING *;

-- name: KitchenSoftDelete :exec
UPDATE kitchens SET deleted_at = now()
WHERE id = $1 AND deleted_at IS NULL;
