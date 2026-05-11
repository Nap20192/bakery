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
    created_at
) VALUES (
    sqlc.arg(number),
    sqlc.arg(location),
    sqlc.narg(from_department_id),
    sqlc.narg(to_department_id),
    sqlc.arg(created_at)
)
RETURNING *;

-- name: CreateOrderItem :one
INSERT INTO order_items (
    order_id,
    iiko_product_id,
    product_name,
    quantity
) VALUES (
    sqlc.arg(order_id),
    sqlc.narg(iiko_product_id),
    sqlc.arg(product_name),
    sqlc.arg(quantity)
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
    COALESCE(p.code, '') AS product_code
FROM order_items AS oi
LEFT JOIN iiko_products AS p ON p.id = oi.iiko_product_id
WHERE oi.order_id = sqlc.arg(order_id)
ORDER BY oi.id;

-- name: ListOrders :many
SELECT *
FROM orders
ORDER BY id DESC
LIMIT sqlc.arg(order_limit);
