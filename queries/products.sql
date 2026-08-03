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

-- name: UpsertDishCatalogItem :one
-- category_id keeps an already-assigned category on conflict, so re-seeding
-- from templates/dishes.txt never clobbers the admin's assignment.
INSERT INTO dish_catalog (
    code,
    name,
    group_id,
    category_id,
    sort_order,
    created_at,
    updated_at
) VALUES (
    sqlc.arg(code),
    sqlc.arg(name),
    sqlc.narg(group_id),
    sqlc.narg(category_id),
    sqlc.arg(sort_order),
    sqlc.arg(created_at),
    sqlc.arg(updated_at)
)
ON CONFLICT (code) DO UPDATE SET
    name = excluded.name,
    group_id = COALESCE(excluded.group_id, dish_catalog.group_id),
    category_id = COALESCE(dish_catalog.category_id, excluded.category_id),
    sort_order = excluded.sort_order,
    updated_at = excluded.updated_at
RETURNING *;

-- name: SearchIikoDishes :many
SELECT id, code, name, measure_unit
FROM iiko_products
WHERE type = 'DISH'
  AND trim(code) <> ''
  AND (name ILIKE '%' || sqlc.arg(query) || '%' OR code ILIKE '%' || sqlc.arg(query) || '%')
ORDER BY name, code
LIMIT sqlc.arg(lim);

-- name: SetDishCatalogSortOrder :exec
UPDATE dish_catalog SET
    sort_order = sqlc.arg(sort_order),
    updated_at = sqlc.arg(updated_at)
WHERE code = sqlc.arg(code);

-- name: UpdateDishCatalogItem :one
UPDATE dish_catalog SET
    code = sqlc.arg(new_code),
    name = sqlc.arg(name),
    group_id = sqlc.narg(group_id),
    category_id = sqlc.narg(category_id),
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
ORDER BY CASE WHEN sort_order = 0 THEN 1 ELSE 0 END, sort_order, id, name, code;

-- name: ListDishCatalogItemsByName :many
SELECT *
FROM dish_catalog
WHERE lower(trim(name)) = lower(trim(sqlc.arg(name)))
ORDER BY CASE WHEN sort_order = 0 THEN 1 ELSE 0 END, sort_order, id, name, code;

-- name: ListDishGroups :many
SELECT id, category_id, name, sort_order
FROM dish_groups
ORDER BY sort_order, id;

-- name: EnsureDishGroup :one
-- Find-or-create a group by (category, name); returns its id. The repo resolves
-- a dish's group name to this id before upserting the dish.
-- ponytail: UNIQUE(category_id, name) does not dedupe NULL-category groups
-- (Postgres treats NULLs as distinct), so uncategorized groups can duplicate.
-- Harmless — the UI groups by name — upgrade to NULLS NOT DISTINCT (PG15+) if it matters.
INSERT INTO dish_groups (category_id, name)
VALUES (sqlc.narg(category_id), sqlc.arg(name))
ON CONFLICT (category_id, name) DO UPDATE SET updated_at = now()
RETURNING id;
