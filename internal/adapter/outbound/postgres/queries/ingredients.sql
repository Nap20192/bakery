-- name: IngredientUpsert :one
INSERT INTO ingredients (iiko_id, name, unit, is_dough)
VALUES ($1, $2, $3, $4)
ON CONFLICT (iiko_id) DO UPDATE
    SET name = EXCLUDED.name,
        unit = EXCLUDED.unit,
        is_dough = EXCLUDED.is_dough,
        updated_at = now(),
        deleted_at = NULL
RETURNING *;

-- name: IngredientGetByID :one
SELECT * FROM ingredients
WHERE id = $1 AND deleted_at IS NULL;

-- name: IngredientGetByIikoID :one
SELECT * FROM ingredients
WHERE iiko_id = $1 AND deleted_at IS NULL;

-- name: IngredientListDough :many
SELECT * FROM ingredients
WHERE is_dough = true AND deleted_at IS NULL
ORDER BY name;

-- name: IngredientList :many
SELECT * FROM ingredients
WHERE deleted_at IS NULL
ORDER BY name;
