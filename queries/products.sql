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

-- name: UpsertDishCatalogItem :one
INSERT INTO dish_catalog (
    code,
    name,
    theme,
    sort_order,
    created_at,
    updated_at
) VALUES (
    sqlc.arg(code),
    sqlc.arg(name),
    sqlc.arg(theme),
    sqlc.arg(sort_order),
    sqlc.arg(created_at),
    sqlc.arg(updated_at)
)
ON CONFLICT (code) DO UPDATE SET
    name = excluded.name,
    theme = excluded.theme,
    sort_order = excluded.sort_order,
    updated_at = excluded.updated_at
RETURNING *;

-- name: UpdateDishCatalogItem :one
UPDATE dish_catalog SET
    code = sqlc.arg(new_code),
    name = sqlc.arg(name),
    theme = sqlc.arg(theme),
    sort_order = sqlc.arg(sort_order),
    updated_at = sqlc.arg(updated_at)
WHERE code = sqlc.arg(code)
RETURNING *;

-- name: DeleteDishCatalogItem :exec
DELETE FROM dish_catalog
WHERE code = sqlc.arg(code);

-- name: ListDishCatalogItems :many
SELECT *
FROM dish_catalog
ORDER BY CASE WHEN sort_order = 0 THEN 1 ELSE 0 END, sort_order, id, theme, name, code;

-- name: ListDishCatalogItemsByName :many
SELECT *
FROM dish_catalog
WHERE lower(trim(name)) = lower(trim(sqlc.arg(name)))
ORDER BY CASE WHEN sort_order = 0 THEN 1 ELSE 0 END, sort_order, id, theme, name, code;
