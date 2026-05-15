-- name: CreateOrderCounterDay :exec
INSERT INTO order_counters(day, counter)
VALUES (sqlc.arg(day), 0)
ON CONFLICT(day) DO NOTHING;

-- name: NextOrderCounter :one
UPDATE order_counters
SET counter = counter + 1
WHERE day = sqlc.arg(day)
RETURNING counter;

-- name: CreateOrder :one
INSERT INTO orders (
    number,
    location,
    from_department_id,
    to_department_id,
    created_at,
    fulfillment_date,
    created_by_username
) VALUES (
    sqlc.arg(number),
    sqlc.arg(location),
    sqlc.narg(from_department_id),
    sqlc.narg(to_department_id),
    sqlc.arg(created_at),
    sqlc.arg(fulfillment_date),
    sqlc.arg(created_by_username)
)
RETURNING *;

-- name: CreateOrderItem :one
INSERT INTO order_items (
    order_id,
    iiko_product_id,
    product_name,
    quantity,
    reserved_quantity
) VALUES (
    sqlc.arg(order_id),
    sqlc.narg(iiko_product_id),
    sqlc.arg(product_name),
    sqlc.arg(quantity),
    sqlc.arg(reserved_quantity)
)
RETURNING *;

-- name: GetOrderByNumber :one
SELECT *
FROM orders
WHERE number = sqlc.arg(number);

-- name: GetOrderItemsByOrderID :many
SELECT
    oi.id,
    oi.order_id,
    oi.iiko_product_id,
    oi.product_name,
    oi.quantity,
    oi.reserved_quantity,
    COALESCE(p.code, '') AS product_code
FROM order_items AS oi
LEFT JOIN iiko_products AS p ON p.id = oi.iiko_product_id
WHERE oi.order_id = sqlc.arg(order_id)
ORDER BY oi.id;

-- name: DeleteOrderItemsByOrderID :exec
DELETE FROM order_items
WHERE order_id = sqlc.arg(order_id);

-- name: UpdateOrder :one
UPDATE orders
SET
    from_department_id = sqlc.narg(from_department_id),
    to_department_id = sqlc.narg(to_department_id),
    fulfillment_date = sqlc.arg(fulfillment_date),
    created_by_username = sqlc.arg(created_by_username)
WHERE number = sqlc.arg(number)
RETURNING *;

-- name: ListOrders :many
SELECT *
FROM orders
ORDER BY id DESC
LIMIT sqlc.arg(order_limit)
OFFSET sqlc.arg(order_offset);

-- name: CountOrders :one
SELECT COUNT(*)
FROM orders;

-- name: DeleteOrdersCreatedBefore :one
WITH deleted AS (
    DELETE FROM orders
    WHERE created_at::timestamptz < sqlc.arg(created_at_before)::timestamptz
    RETURNING id
)
SELECT COUNT(*)::BIGINT
FROM deleted;

-- name: CreateOrderTemplate :one
INSERT INTO order_templates (
    name,
    body,
    created_by_user_id,
    created_at,
    updated_at
) VALUES (
    sqlc.arg(name),
    sqlc.arg(body),
    sqlc.narg(created_by_user_id),
    sqlc.arg(created_at),
    sqlc.arg(updated_at)
)
RETURNING *;

-- name: ListOrderTemplates :many
SELECT *
FROM order_templates
ORDER BY name, id;

-- name: GetOrderTemplateByID :one
SELECT *
FROM order_templates
WHERE id = sqlc.arg(id);

-- name: DeleteOrderTemplateByID :exec
DELETE FROM order_templates
WHERE id = sqlc.arg(id);
