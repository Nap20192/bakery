-- name: ListOrderCategories :many
SELECT *
FROM order_categories
ORDER BY sort_order, id;

-- name: GetOrderCategoryByID :one
SELECT *
FROM order_categories
WHERE id = sqlc.arg(id);

-- name: CreateOrderCategory :one
INSERT INTO order_categories (code, letter, name, color, sort_order, monitor_codes)
VALUES (
    sqlc.arg(code),
    sqlc.arg(letter),
    sqlc.arg(name),
    sqlc.arg(color),
    sqlc.arg(sort_order),
    sqlc.arg(monitor_codes)
)
RETURNING *;

-- name: UpdateOrderCategory :one
UPDATE order_categories
SET letter = sqlc.arg(letter),
    name = sqlc.arg(name),
    color = sqlc.arg(color),
    sort_order = sqlc.arg(sort_order),
    monitor_codes = sqlc.arg(monitor_codes),
    updated_at = now()
WHERE id = sqlc.arg(id)
RETURNING *;

-- name: DeleteOrderCategory :exec
DELETE FROM order_categories
WHERE id = sqlc.arg(id);

-- name: CountDishesByCategoryID :one
SELECT COUNT(*)
FROM dish_catalog
WHERE category_id = sqlc.arg(category_id);
