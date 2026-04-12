-- name: CreateKitchen :one
INSERT INTO kitchens (id, user_id, name)
VALUES ($1, $2, $3)
RETURNING *;

-- name: GetKitchenByID :one
SELECT * FROM kitchens WHERE id = $1;

-- name: GetKitchenByUserID :one
SELECT * FROM kitchens WHERE user_id = $1;

-- name: ListKitchens :many
SELECT * FROM kitchens ORDER BY name;

-- name: UpdateKitchen :one
UPDATE kitchens
SET name = $2,
    updated_at = now()
WHERE id = $1
RETURNING *;

-- name: DeleteKitchen :exec
DELETE FROM kitchens WHERE id = $1;
