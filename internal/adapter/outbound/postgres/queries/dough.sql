-- name: DoughCalcInsert :one
INSERT INTO dough_calculations (kitchen_id, calc_date, dough_ingredient_id, total_qty, unit, source)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: DoughCalcGetByDate :many
SELECT dc.*,
       i.name AS ingredient_name,
       k.name AS kitchen_name
FROM dough_calculations dc
JOIN ingredients i ON i.id = dc.dough_ingredient_id
LEFT JOIN kitchens k ON k.id = dc.kitchen_id
WHERE dc.calc_date = $1
ORDER BY k.name NULLS FIRST, i.name;

-- name: DoughCalcItemInsert :one
INSERT INTO dough_calculation_items
    (calculation_id, order_item_id, product_id, tech_card_id, quantity, dough_qty, unit)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: DoughCalcItemListByCalc :many
SELECT dci.*, p.name AS product_name
FROM dough_calculation_items dci
JOIN products p ON p.id = dci.product_id
WHERE dci.calculation_id = $1
ORDER BY p.name;

-- name: DoughCalcItemDeleteByCalc :exec
DELETE FROM dough_calculation_items WHERE calculation_id = $1;

-- name: DoughCalcComputeTotals :many
SELECT
    tci.ingredient_id  AS dough_ingredient_id,
    i.name             AS dough_name,
    tci.unit           AS dough_unit,
    SUM(oi.quantity * (tci.net_qty / tc.yield_qty)) AS total_dough
FROM order_items oi
JOIN orders o     ON o.id = oi.order_id
JOIN products p   ON p.id = oi.product_id
JOIN LATERAL (
    SELECT id, yield_qty
    FROM tech_cards
    WHERE product_id = oi.product_id
      AND deleted_at IS NULL
      AND valid_from <= o.order_date
      AND (valid_to IS NULL OR valid_to >= o.order_date)
    ORDER BY version DESC
    LIMIT 1
) tc ON true
JOIN tech_card_items tci ON tci.tech_card_id = tc.id
JOIN ingredients i       ON i.id = tci.ingredient_id
                         AND i.is_dough = true
                         AND i.deleted_at IS NULL
WHERE o.order_date = $1
  AND o.status IN ('confirmed', 'parsed')
  AND o.deleted_at IS NULL
GROUP BY tci.ingredient_id, i.name, tci.unit
ORDER BY i.name;
