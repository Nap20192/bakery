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
SET updated_at = strftime('%Y-%m-%d %H:%M:%f+00:00', 'now')
WHERE id = sqlc.arg(id);

-- name: DeleteProductionSheet :exec
DELETE FROM production_sheets
WHERE id = sqlc.arg(id);

-- name: ListProductionSheets :many
-- order_numbers_json is the saved batch selection (production_sheet_orders), not
-- the orders with deviations: a sheet covers its whole batch.
SELECT
    ps.id,
    ps.created_by_username,
    ps.created_at,
    ps.updated_at,
    CAST((
        SELECT COUNT(*)
        FROM production_sheet_items psi
        WHERE psi.sheet_id = ps.id
    ) AS INTEGER) AS item_count,
    CAST((
        SELECT json_group_array(numbers.number)
        FROM (
            SELECT o.number
            FROM production_sheet_orders pso
            JOIN orders o ON o.id = pso.order_id
            WHERE pso.sheet_id = ps.id
            ORDER BY o.number
        ) AS numbers
    ) AS TEXT) AS order_numbers_json
FROM production_sheets ps
ORDER BY ps.id DESC;

-- name: InsertProductionSheetOrder :exec
INSERT INTO production_sheet_orders (sheet_id, order_id)
VALUES (sqlc.arg(sheet_id), sqlc.arg(order_id));

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
    CAST(COALESCE(psi.produced_quantity, oi.quantity + oi.reserved_quantity) AS REAL) AS produced_quantity,
    CAST(COALESCE(psi.reason, '') AS TEXT) AS reason
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
-- Read-time decorator: production fact for order items from the journal (the
-- newest sheet wins per item). The key is the normalized product name.
SELECT
    ranked.order_id,
    CAST(ranked.product_key AS TEXT) AS product_key,
    CAST(ranked.produced_quantity AS REAL) AS produced_quantity,
    CAST(ranked.reason AS TEXT) AS reason
FROM (
    SELECT
        psi.order_id,
        lower(trim(psi.product_name)) AS product_key,
        psi.produced_quantity,
        psi.reason,
        ROW_NUMBER() OVER (
            PARTITION BY psi.order_id, lower(trim(psi.product_name))
            ORDER BY psi.sheet_id DESC
        ) AS rn
    FROM production_sheet_items psi
    WHERE psi.order_id IN (sqlc.slice(order_ids))
) AS ranked
WHERE ranked.rn = 1
ORDER BY ranked.order_id, ranked.product_key;

-- name: GetOrderProductionSheetID :one
-- An order belongs to at most one production sheet (1:N from sheet to
-- orders); the newest sheet is taken in case of historical overlaps.
-- Membership is defined by the saved selection, not by deviations.
SELECT sheet_id
FROM production_sheet_orders
WHERE order_id = sqlc.arg(order_id)
ORDER BY sheet_id DESC
LIMIT 1;

-- name: ListOrderProductionSheetIDs :many
-- Batch version for order lists: the matrix must see production membership
-- without a separate query per card.
SELECT
    order_id,
    CAST(MAX(sheet_id) AS INTEGER) AS sheet_id
FROM production_sheet_orders
WHERE order_id IN (sqlc.slice(order_ids))
GROUP BY order_id
ORDER BY order_id;
