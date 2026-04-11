-- name: OrderInsert :one
INSERT INTO orders (client_id, kitchen_id, telegram_chat_id, telegram_message_id,
                    order_date, status, raw_text, note)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING *;

-- name: OrderGetByID :one
SELECT * FROM orders
WHERE id = $1 AND deleted_at IS NULL;

-- name: OrderListByDate :many
SELECT * FROM orders
WHERE order_date = $1 AND deleted_at IS NULL
ORDER BY created_at;

-- name: OrderListByDateAndStatus :many
SELECT * FROM orders
WHERE order_date = $1 AND status = $2 AND deleted_at IS NULL
ORDER BY created_at;

-- name: OrderUpdateStatus :one
UPDATE orders SET status = $2
WHERE id = $1 AND deleted_at IS NULL
RETURNING *;

-- name: OrderSoftDelete :exec
UPDATE orders SET deleted_at = now(), status = 'cancelled'
WHERE id = $1 AND deleted_at IS NULL;

-- name: OrderItemInsert :one
INSERT INTO order_items (order_id, product_id, quantity, unit)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: OrderItemListByOrder :many
SELECT oi.*, p.name AS product_name
FROM order_items oi
JOIN products p ON p.id = oi.product_id
WHERE oi.order_id = $1
ORDER BY p.name;

-- name: OrderItemDeleteByOrder :exec
DELETE FROM order_items WHERE order_id = $1;
