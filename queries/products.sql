-- name: DishExistsByCode :one
SELECT COUNT(*)::bigint
FROM iiko_products
WHERE code = sqlc.arg(code)
  AND type = 'DISH';

-- name: GetIikoProductByCode :one
SELECT
    id,
    code,
    name,
    type,
    measure_unit,
    raw_json
FROM iiko_products
WHERE code = sqlc.arg(code)
ORDER BY
    CASE WHEN type = 'DISH' THEN 0 ELSE 1 END,
    updated_at DESC,
    id
LIMIT 1;

-- name: GetIikoProductByID :one
SELECT
    id,
    code,
    name,
    type,
    measure_unit,
    raw_json
FROM iiko_products
WHERE id = sqlc.arg(id);

-- name: GetIikoProductsByName :many
SELECT
    id,
    code,
    name,
    type,
    measure_unit,
    raw_json
FROM iiko_products
WHERE trim(name) = trim(sqlc.arg(name));
