-- name: GetActiveAssemblyChartByProductID :one
SELECT
    id,
    assembled_product_id,
    assembled_amount,
    date_from,
    date_to
FROM iiko_assembly_charts
WHERE assembled_product_id = sqlc.arg(assembled_product_id)
  AND date_from::date <= sqlc.arg(order_date)::date
  AND (date_to IS NULL OR date_to::date >= sqlc.arg(order_date)::date)
ORDER BY date_from::date DESC
LIMIT 1;

-- name: GetActiveAssemblyChartFullByProductID :one
SELECT *
FROM iiko_assembly_charts
WHERE assembled_product_id = sqlc.arg(assembled_product_id)
  AND date_from::date <= sqlc.arg(order_date)::date
  AND (date_to IS NULL OR date_to::date >= sqlc.arg(order_date)::date)
ORDER BY date_from::date DESC
LIMIT 1;

-- name: ListAssemblyChartItemsByChartID :many
SELECT
    i.id AS item_id,
    i.product_id,
    i.amount_in,
    i.amount_out,
    COALESCE(p.name, '') AS product_name,
    COALESCE(p.code, '') AS product_code,
    COALESCE(p.measure_unit, '') AS measure_unit
FROM iiko_assembly_chart_items AS i
LEFT JOIN iiko_products AS p ON p.id = i.product_id
WHERE i.chart_id = sqlc.arg(chart_id)
ORDER BY i.sort_weight, i.id;

-- name: ListPreparedChartItemsByChartID :many
SELECT
    i.id AS item_id,
    i.product_id,
    i.amount,
    COALESCE(p.name, '') AS product_name,
    COALESCE(p.code, '') AS product_code,
    COALESCE(p.measure_unit, '') AS measure_unit
FROM iiko_prepared_chart_items AS i
LEFT JOIN iiko_products AS p ON p.id = i.product_id
WHERE i.prepared_chart_id = sqlc.arg(prepared_chart_id)
ORDER BY i.sort_weight, i.id;

-- name: GetActivePreparedChartFullByProductID :one
SELECT *
FROM iiko_prepared_charts
WHERE assembled_product_id = sqlc.arg(assembled_product_id)
  AND date_from::date <= sqlc.arg(order_date)::date
  AND (date_to IS NULL OR date_to::date >= sqlc.arg(order_date)::date)
ORDER BY date_from::date DESC
LIMIT 1;
