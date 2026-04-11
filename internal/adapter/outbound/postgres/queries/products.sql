-- name: ProductUpsert :one
INSERT INTO products (iiko_id, name, unit)
VALUES ($1, $2, $3)
ON CONFLICT (iiko_id) DO UPDATE
    SET name = EXCLUDED.name,
        unit = EXCLUDED.unit,
        updated_at = now(),
        deleted_at = NULL
RETURNING *;

-- name: ProductGetByID :one
SELECT * FROM products
WHERE id = $1 AND deleted_at IS NULL;

-- name: ProductGetByIikoID :one
SELECT * FROM products
WHERE iiko_id = $1 AND deleted_at IS NULL;

-- name: ProductList :many
SELECT * FROM products
WHERE deleted_at IS NULL
ORDER BY name;
