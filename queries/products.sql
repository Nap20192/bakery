-- name: GetProducts :many
SELECT
    id,
    code,
    name,
    type,
    measure_unit,
    raw_json
FROM iiko_products
WHERE type = COALESCE(sqlc.narg('type'), type);
