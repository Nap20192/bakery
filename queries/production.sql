-- name: CreateProductionSheet :one
INSERT INTO production_sheets (created_by_username)
VALUES (sqlc.arg(created_by_username))
RETURNING *;

-- name: GetProductionSheet :one
SELECT *
FROM production_sheets
WHERE id = sqlc.arg(id);

-- name: TouchProductionSheet :exec
UPDATE production_sheets
SET updated_at = now()
WHERE id = sqlc.arg(id);

-- name: DeleteProductionSheet :exec
DELETE FROM production_sheets
WHERE id = sqlc.arg(id);

-- name: ListProductionSheets :many
-- order_numbers — сохранённый выбор партии (production_sheet_orders), а не
-- заказы с отклонениями: лист покрывает партию целиком.
SELECT
    ps.id,
    ps.created_by_username,
    ps.created_at,
    ps.updated_at,
    (
        SELECT COUNT(*)
        FROM production_sheet_items psi
        WHERE psi.sheet_id = ps.id
    )::BIGINT AS item_count,
    COALESCE((
        SELECT ARRAY_AGG(o.number ORDER BY o.number)
        FROM production_sheet_orders pso
        JOIN orders o ON o.id = pso.order_id
        WHERE pso.sheet_id = ps.id
    ), '{}')::TEXT[] AS order_numbers
FROM production_sheets ps
ORDER BY ps.id DESC;

-- name: InsertProductionSheetOrder :exec
INSERT INTO production_sheet_orders (sheet_id, order_id)
VALUES (sqlc.arg(sheet_id), sqlc.arg(order_id))
ON CONFLICT DO NOTHING;

-- name: DeleteProductionSheetOrders :exec
DELETE FROM production_sheet_orders
WHERE sheet_id = sqlc.arg(sheet_id);

-- name: ListProductionSheetOrderNumbers :many
SELECT o.number
FROM production_sheet_orders pso
JOIN orders o ON o.id = pso.order_id
WHERE pso.sheet_id = sqlc.arg(sheet_id)
ORDER BY o.number;

-- name: InsertProductionSheetItem :exec
INSERT INTO production_sheet_items (sheet_id, order_id, product_name, produced_quantity, reason)
VALUES (sqlc.arg(sheet_id), sqlc.arg(order_id), sqlc.arg(product_name), sqlc.arg(produced_quantity), sqlc.arg(reason));

-- name: InsertProductionSheetLoad :exec
INSERT INTO production_sheet_loads (sheet_id, order_id, product_name, loaded_quantity)
VALUES (sqlc.arg(sheet_id), sqlc.arg(order_id), sqlc.arg(product_name), sqlc.arg(loaded_quantity));

-- name: DeleteProductionSheetItems :exec
DELETE FROM production_sheet_items
WHERE sheet_id = sqlc.arg(sheet_id);

-- name: DeleteProductionSheetLoads :exec
DELETE FROM production_sheet_loads
WHERE sheet_id = sqlc.arg(sheet_id);

-- name: ListProductionSheetItems :many
SELECT
    psl.order_id,
    o.number AS order_number,
    psl.product_name,
    psl.loaded_quantity,
    COALESCE(psi.produced_quantity, oi.quantity + oi.reserved_quantity)::DOUBLE PRECISION AS produced_quantity,
    COALESCE(psi.reason, '')::TEXT AS reason
FROM production_sheet_loads psl
JOIN orders o ON o.id = psl.order_id
JOIN order_items oi
    ON oi.order_id = psl.order_id
    AND lower(trim(oi.product_name)) = lower(trim(psl.product_name))
LEFT JOIN production_sheet_items psi
    ON psi.sheet_id = psl.sheet_id
    AND psi.order_id = psl.order_id
    AND lower(trim(psi.product_name)) = lower(trim(psl.product_name))
WHERE psl.sheet_id = sqlc.arg(sheet_id)
ORDER BY psl.product_name, o.number;

-- name: ListProductionSheetOrderIDs :many
SELECT order_id
FROM production_sheet_orders
WHERE sheet_id = sqlc.arg(sheet_id);

-- name: GetOrderProductionFacts :many
-- Read-time декоратор: факт выпечки для позиций заказов из журнала (по
-- каждой позиции побеждает самый свежий лист). Ключ — нормализованное имя.
SELECT DISTINCT ON (psi.order_id, lower(trim(psi.product_name)))
    psi.order_id,
    lower(trim(psi.product_name)) AS product_key,
    psi.produced_quantity,
    psi.reason
FROM production_sheet_items psi
WHERE psi.order_id = ANY(sqlc.arg(order_ids)::BIGINT[])
ORDER BY psi.order_id, lower(trim(psi.product_name)), psi.sheet_id DESC;

-- name: GetOrderProductionSheetID :one
-- Заказ принадлежит максимум одному листу отработки (1:N от листа к
-- заказам); на случай исторических пересечений берётся самый свежий лист.
-- Принадлежность определяется сохранённым выбором, не отклонениями.
SELECT sheet_id
FROM production_sheet_orders
WHERE order_id = sqlc.arg(order_id)
ORDER BY sheet_id DESC
LIMIT 1;

-- name: ListOrderProductionSheetIDs :many
-- Пакетная версия для списка заказов: матрица должна видеть принадлежность
-- к отработке без отдельного запроса на каждую карточку.
SELECT DISTINCT ON (order_id)
    order_id,
    sheet_id
FROM production_sheet_orders
WHERE order_id = ANY(sqlc.arg(order_ids)::BIGINT[])
ORDER BY order_id, sheet_id DESC;
