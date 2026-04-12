-- name: CreateOrder :one
INSERT INTO orders (
    id, client_id, kitchen_id, telegram_chat_id, telegram_message_id,
    order_date, status, raw_text, note
) VALUES ($1, $2, $3, $4, $5, $6, COALESCE(sqlc.narg('status')::text, 'new'), $7, $8)
RETURNING *;

-- name: GetOrderByID :one
SELECT * FROM orders WHERE id = $1;

-- name: ListOrders :many
SELECT * FROM orders
WHERE (sqlc.narg('status')::text IS NULL OR status = sqlc.narg('status')::text)
ORDER BY order_date DESC, created_at DESC;

-- name: ListOrdersByClient :many
SELECT * FROM orders
WHERE client_id = $1
  AND (sqlc.narg('status')::text IS NULL OR status = sqlc.narg('status')::text)
ORDER BY order_date DESC, created_at DESC;

-- name: ListOrdersByKitchen :many
SELECT * FROM orders
WHERE kitchen_id = $1
  AND (sqlc.narg('status')::text IS NULL OR status = sqlc.narg('status')::text)
ORDER BY order_date DESC, created_at DESC;

-- name: ListOrdersByDate :many
SELECT * FROM orders
WHERE order_date = $1
  AND (sqlc.narg('status')::text IS NULL OR status = sqlc.narg('status')::text)
ORDER BY created_at DESC;

-- name: ListOrdersByDateRange :many
SELECT * FROM orders
WHERE order_date BETWEEN $1 AND $2
  AND (sqlc.narg('status')::text IS NULL OR status = sqlc.narg('status')::text)
ORDER BY order_date, created_at;

-- name: UpdateOrderStatus :one
UPDATE orders
SET status = $2,
    updated_at = now()
WHERE id = $1
RETURNING *;

-- name: UpdateOrder :one
UPDATE orders
SET kitchen_id = $2,
    order_date = $3,
    status = $4,
    note = $5,
    updated_at = now()
WHERE id = $1
RETURNING *;

-- name: DeleteOrder :exec
DELETE FROM orders WHERE id = $1;

-- name: CreateOrderItem :one
INSERT INTO order_items (id, order_id, product_id, quantity, unit)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: ListOrderItems :many
SELECT * FROM order_items
WHERE order_id = $1;

-- name: DeleteOrderItems :exec
DELETE FROM order_items WHERE order_id = $1;
