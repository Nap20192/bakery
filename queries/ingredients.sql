-- name: CreateIngredient :one
INSERT INTO ingredients (id, iiko_id, name, unit)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: UpsertIngredientByIikoID :one
INSERT INTO ingredients (id, iiko_id, name, unit)
VALUES ($1, $2, $3, $4)
ON CONFLICT (iiko_id) DO UPDATE
SET name = EXCLUDED.name,
    unit = EXCLUDED.unit,
    updated_at = now()
RETURNING *;

-- name: GetIngredientByID :one
SELECT * FROM ingredients WHERE id = $1;

-- name: GetIngredientByIikoID :one
SELECT * FROM ingredients WHERE iiko_id = $1;

-- name: GetIngredientByName :one
SELECT * FROM ingredients WHERE name = $1;

-- name: ListIngredients :many
SELECT * FROM ingredients ORDER BY name;

-- name: DeleteIngredient :exec
DELETE FROM ingredients WHERE id = $1;
