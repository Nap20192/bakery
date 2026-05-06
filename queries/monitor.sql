-- name: GetActiveAssemblyChartByProductID :one
SELECT
    id,
    assembled_product_id,
    assembled_amount,
    date_from,
    date_to
FROM iiko_assembly_charts
WHERE assembled_product_id = sqlc.arg(assembled_product_id)
  AND date(date_from) <= date(sqlc.arg(order_date))
  AND (date_to IS NULL OR date(date_to) >= date(sqlc.arg(order_date)))
ORDER BY date(date_from) DESC
LIMIT 1;

-- name: ListAssemblyChartItemsByChartID :many
SELECT
    i.id AS item_id,
    i.product_id,
    i.amount_out,
    COALESCE(p.name, '') AS product_name,
    COALESCE(p.code, '') AS product_code,
    COALESCE(p.measure_unit, '') AS measure_unit
FROM iiko_assembly_chart_items AS i
LEFT JOIN iiko_products AS p ON p.id = i.product_id
WHERE i.chart_id = sqlc.arg(chart_id)
ORDER BY i.sort_weight, i.id;
