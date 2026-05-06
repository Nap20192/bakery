-- name: GetProducts :many
SELECT
    id,
    code,
    name,
    type,
    measure_unit,
    raw_json
FROM iiko_products
WHERE type = COALESCE(sqlc.narg(type), type);

-- name: DishExistsByCode :one
SELECT EXISTS (
    SELECT 1
    FROM iiko_products
    WHERE code = sqlc.arg(code)
      AND type = 'DISH'
);

-- name: GetIikoProductByCode :one
SELECT
    id,
    code,
    name,
    type,
    measure_unit,
    raw_json
FROM iiko_products
WHERE code = sqlc.arg(code);

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

-- name: ListIikoProductsByGroupID :many
SELECT
    id,
    code,
    name,
    type,
    measure_unit,
    raw_json
FROM iiko_products
WHERE lower(json_extract(raw_json, '$.groupId')) = lower(sqlc.arg(group_id));
