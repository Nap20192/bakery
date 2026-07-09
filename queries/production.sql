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
SELECT
    ps.id,
    ps.created_by_username,
    ps.created_at,
    ps.updated_at,
    COUNT(psi.id)::BIGINT AS item_count,
    COALESCE(ARRAY_AGG(DISTINCT o.number) FILTER (WHERE o.number IS NOT NULL), '{}')::TEXT[] AS order_numbers
FROM production_sheets ps
LEFT JOIN production_sheet_items psi ON psi.sheet_id = ps.id
LEFT JOIN orders o ON o.id = psi.order_id
GROUP BY ps.id
ORDER BY ps.id DESC;

-- name: InsertProductionSheetItem :exec
INSERT INTO production_sheet_items (sheet_id, order_id, product_name, produced_quantity, reason)
VALUES (sqlc.arg(sheet_id), sqlc.arg(order_id), sqlc.arg(product_name), sqlc.arg(produced_quantity), sqlc.arg(reason));

-- name: DeleteProductionSheetItems :exec
DELETE FROM production_sheet_items
WHERE sheet_id = sqlc.arg(sheet_id);

-- name: ListProductionSheetItems :many
SELECT
    psi.id,
    psi.order_id,
    o.number AS order_number,
    psi.product_name,
    psi.produced_quantity,
    psi.reason
FROM production_sheet_items psi
JOIN orders o ON o.id = psi.order_id
WHERE psi.sheet_id = sqlc.arg(sheet_id)
ORDER BY psi.product_name, o.number;

-- name: ListProductionSheetOrderIDs :many
SELECT DISTINCT order_id
FROM production_sheet_items
WHERE sheet_id = sqlc.arg(sheet_id);

-- name: GetOrderProductionSheetID :one
-- Заказ принадлежит максимум одному листу отработки (1:N от листа к
-- заказам); на случай исторических пересечений берётся самый свежий лист.
SELECT sheet_id
FROM production_sheet_items
WHERE order_id = sqlc.arg(order_id)
ORDER BY sheet_id DESC
LIMIT 1;

-- name: ApplyOrderProduction :exec
-- Проецирует журнал на заказ: для каждой позиции берётся значение из самого
-- свежего листа (по id листа), позиции без записей в журнале сбрасываются.
UPDATE order_items oi
SET produced_quantity = latest.produced,
    produced_reason = NULLIF(latest.reason, '')
FROM (
    SELECT DISTINCT ON (lower(trim(psi.product_name)))
        lower(trim(psi.product_name)) AS pname,
        psi.produced_quantity AS produced,
        psi.reason AS reason
    FROM production_sheet_items psi
    WHERE psi.order_id = sqlc.arg(order_id)
    ORDER BY lower(trim(psi.product_name)), psi.sheet_id DESC
) latest
WHERE oi.order_id = sqlc.arg(order_id)
  AND lower(trim(oi.product_name)) = latest.pname;

-- name: ClearUncoveredOrderProduction :exec
UPDATE order_items oi
SET produced_quantity = NULL,
    produced_reason = NULL
WHERE oi.order_id = sqlc.arg(order_id)
  AND NOT EXISTS (
      SELECT 1
      FROM production_sheet_items psi
      WHERE psi.order_id = oi.order_id
        AND lower(trim(psi.product_name)) = lower(trim(oi.product_name))
  );
