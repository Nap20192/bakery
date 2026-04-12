-- name: CreateProduct :one
INSERT INTO products (id, iiko_id, name, unit)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: UpsertProductByIikoID :one
INSERT INTO products (id, iiko_id, name, unit)
VALUES ($1, $2, $3, $4)
ON CONFLICT (iiko_id) DO UPDATE
SET name = EXCLUDED.name,
    unit = EXCLUDED.unit,
    updated_at = now()
RETURNING *;

-- name: GetProductByID :one
SELECT * FROM products WHERE id = $1;

-- name: GetProductByIikoID :one
SELECT * FROM products WHERE iiko_id = $1;

-- name: GetProductByName :one
SELECT * FROM products WHERE name = $1;

-- name: ListProducts :many
SELECT * FROM products ORDER BY name;

-- name: DeleteProduct :exec
DELETE FROM products WHERE id = $1;
