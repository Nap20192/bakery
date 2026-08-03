-- name: CreateOrderCounterDay :exec
INSERT INTO order_counters(day, department_id, category_id, counter)
VALUES (sqlc.arg(day), sqlc.arg(department_id), sqlc.arg(category_id), 0)
ON CONFLICT(day, department_id, category_id) DO NOTHING;

-- name: NextOrderCounter :one
UPDATE order_counters
SET counter = counter + 1
WHERE day = sqlc.arg(day) AND department_id = sqlc.arg(department_id) AND category_id = sqlc.arg(category_id)
RETURNING counter;

-- name: CreateOrder :one
INSERT INTO orders (
    number,
    location,
    from_department_id,
    to_department_id,
    category_id,
    created_at,
    fulfillment_date,
    created_by_username,
    comments
) VALUES (
    sqlc.arg(number),
    sqlc.arg(location),
    sqlc.narg(from_department_id),
    sqlc.narg(to_department_id),
    sqlc.narg(category_id),
    sqlc.arg(created_at),
    sqlc.arg(fulfillment_date),
    sqlc.arg(created_by_username),
    sqlc.narg(comments)
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
-- Позиции хранят только заявку. Факт отработки живёт в журнале и
-- декорируется при чтении (GetOrderProductionFacts + мердж в репозитории).
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

-- name: GetOrderItemsByOrderIDs :many
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
WHERE oi.order_id = ANY(sqlc.arg(order_ids)::BIGINT[])
ORDER BY oi.order_id, oi.id;

-- name: GetOrderByID :one
SELECT *
FROM orders
WHERE id = sqlc.arg(id);

-- name: DeleteOrderItemsByOrderID :exec
DELETE FROM order_items
WHERE order_id = sqlc.arg(order_id);

-- name: CreateOrderHistory :one
INSERT INTO order_history (
    order_id,
    changed_by_username,
    changed_at
) VALUES (
    sqlc.arg(order_id),
    sqlc.arg(changed_by_username),
    sqlc.arg(changed_at)
)
RETURNING *;

-- name: CreateOrderHistoryItem :one
INSERT INTO order_history_items (
    history_id,
    change_type,
    product_code,
    product_name,
    old_quantity,
    new_quantity,
    old_reserved_quantity,
    new_reserved_quantity
) VALUES (
    sqlc.arg(history_id),
    sqlc.arg(change_type),
    sqlc.arg(product_code),
    sqlc.arg(product_name),
    sqlc.narg(old_quantity),
    sqlc.narg(new_quantity),
    sqlc.narg(old_reserved_quantity),
    sqlc.narg(new_reserved_quantity)
)
RETURNING *;

-- name: ListOrderHistoryByOrderID :many
SELECT *
FROM order_history
WHERE order_id = sqlc.arg(order_id)
ORDER BY changed_at DESC, id DESC;

-- name: ListOrderHistoryItemsByHistoryID :many
SELECT *
FROM order_history_items
WHERE history_id = sqlc.arg(history_id)
ORDER BY id;

-- name: UpdateOrder :one
-- created_by_username is intentionally left untouched: editing an order keeps
-- its original author; who made each change is recorded in order_history.
UPDATE orders
SET
    from_department_id = sqlc.narg(from_department_id),
    to_department_id = sqlc.narg(to_department_id),
    fulfillment_date = sqlc.arg(fulfillment_date),
    comments = sqlc.narg(comments)
WHERE number = sqlc.arg(number)
RETURNING *;

-- name: ListOrders :many
SELECT *
FROM orders
WHERE
    (sqlc.narg(from_department_id)::BIGINT IS NULL OR from_department_id = sqlc.narg(from_department_id)::BIGINT)
    AND (sqlc.narg(fulfillment_date)::DATE IS NULL OR fulfillment_date = sqlc.narg(fulfillment_date)::DATE)
    AND (sqlc.narg(fulfillment_from)::DATE IS NULL OR fulfillment_date >= sqlc.narg(fulfillment_from)::DATE)
    AND (sqlc.narg(fulfillment_to)::DATE IS NULL OR fulfillment_date <= sqlc.narg(fulfillment_to)::DATE)
    AND (sqlc.narg(category_id)::BIGINT IS NULL OR category_id = sqlc.narg(category_id)::BIGINT)
ORDER BY id DESC
LIMIT sqlc.arg(order_limit)
OFFSET sqlc.arg(order_offset);

-- name: CountOrders :one
SELECT COUNT(*)
FROM orders
WHERE
    (sqlc.narg(from_department_id)::BIGINT IS NULL OR from_department_id = sqlc.narg(from_department_id)::BIGINT)
    AND (sqlc.narg(fulfillment_date)::DATE IS NULL OR fulfillment_date = sqlc.narg(fulfillment_date)::DATE)
    AND (sqlc.narg(fulfillment_from)::DATE IS NULL OR fulfillment_date >= sqlc.narg(fulfillment_from)::DATE)
    AND (sqlc.narg(fulfillment_to)::DATE IS NULL OR fulfillment_date <= sqlc.narg(fulfillment_to)::DATE)
    AND (sqlc.narg(category_id)::BIGINT IS NULL OR category_id = sqlc.narg(category_id)::BIGINT);

-- name: SetOrderFavorite :one
UPDATE orders
SET is_favorite = sqlc.arg(is_favorite)
WHERE number = sqlc.arg(number)
RETURNING *;

-- name: CancelOrder :one
UPDATE orders
SET cancelled_at = sqlc.arg(cancelled_at),
    cancelled_by_username = sqlc.arg(cancelled_by_username)
WHERE number = sqlc.arg(number)
RETURNING *;

-- name: RestoreOrder :one
UPDATE orders
SET cancelled_at = NULL,
    cancelled_by_username = ''
WHERE number = sqlc.arg(number)
RETURNING *;

-- name: DeleteOrdersCreatedBefore :one
WITH deleted AS (
    DELETE FROM orders
    WHERE created_at < sqlc.arg(created_at_before)
      AND is_favorite = FALSE
    RETURNING id
)
SELECT COUNT(*)::BIGINT
FROM deleted;
