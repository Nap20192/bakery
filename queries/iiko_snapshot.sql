-- name: CreateIikoSyncRun :one
INSERT INTO iiko_sync_runs (
    source,
    date_from,
    date_to,
    known_revision,
    status,
    error,
    started_at,
    finished_at
) VALUES (
    sqlc.arg(source),
    sqlc.arg(date_from),
    sqlc.arg(date_to),
    sqlc.arg(known_revision),
    sqlc.arg(status),
    sqlc.arg(error),
    sqlc.arg(started_at),
    sqlc.narg(finished_at)
)
RETURNING *;

-- name: FinishIikoSyncRun :one
UPDATE iiko_sync_runs
SET
    known_revision = sqlc.arg(known_revision),
    status = sqlc.arg(status),
    error = sqlc.arg(error),
    finished_at = sqlc.narg(finished_at)
WHERE id = sqlc.arg(id)
RETURNING *;

-- name: UpsertIikoProduct :one
INSERT INTO iiko_products (
    id,
    code,
    name,
    type,
    measure_unit,
    raw_json,
    updated_at
) VALUES (
    sqlc.arg(id),
    sqlc.arg(code),
    sqlc.arg(name),
    sqlc.narg(type),
    sqlc.arg(measure_unit),
    sqlc.arg(raw_json),
    sqlc.arg(updated_at)
)
ON CONFLICT(code) DO UPDATE SET
    name = excluded.name,
    type = excluded.type,
    measure_unit = excluded.measure_unit,
    raw_json = excluded.raw_json,
    updated_at = excluded.updated_at
RETURNING *;

-- name: UpsertIikoAssemblyChart :one
INSERT INTO iiko_assembly_charts (
    id,
    assembled_product_id,
    date_from,
    date_to,
    assembled_amount,
    product_writeoff_strategy,
    product_size_assembly_strategy,
    effective_direct_writeoff_store_spec_json,
    raw_json,
    updated_at
) VALUES (
    sqlc.arg(id),
    sqlc.arg(assembled_product_id),
    sqlc.arg(date_from),
    sqlc.narg(date_to),
    sqlc.arg(assembled_amount),
    sqlc.arg(product_writeoff_strategy),
    sqlc.arg(product_size_assembly_strategy),
    sqlc.arg(effective_direct_writeoff_store_spec_json),
    sqlc.arg(raw_json),
    sqlc.arg(updated_at)
)
ON CONFLICT(id) DO UPDATE SET
    assembled_product_id = excluded.assembled_product_id,
    date_from = excluded.date_from,
    date_to = excluded.date_to,
    assembled_amount = excluded.assembled_amount,
    product_writeoff_strategy = excluded.product_writeoff_strategy,
    product_size_assembly_strategy = excluded.product_size_assembly_strategy,
    effective_direct_writeoff_store_spec_json = excluded.effective_direct_writeoff_store_spec_json,
    raw_json = excluded.raw_json,
    updated_at = excluded.updated_at
RETURNING *;

-- name: DeleteIikoAssemblyChartItemsByChartID :exec
DELETE FROM iiko_assembly_chart_items
WHERE chart_id = sqlc.arg(chart_id);

-- name: InsertIikoAssemblyChartItem :one
INSERT INTO iiko_assembly_chart_items (
    id,
    chart_id,
    sort_weight,
    product_id,
    product_size_spec_json,
    store_spec_json,
    amount_in,
    amount_middle,
    amount_out,
    amount_in1,
    amount_out1,
    amount_in2,
    amount_out2,
    amount_in3,
    amount_out3,
    package_count,
    package_type_id,
    raw_json
) VALUES (
    sqlc.arg(id),
    sqlc.arg(chart_id),
    sqlc.arg(sort_weight),
    sqlc.arg(product_id),
    sqlc.arg(product_size_spec_json),
    sqlc.arg(store_spec_json),
    sqlc.arg(amount_in),
    sqlc.arg(amount_middle),
    sqlc.arg(amount_out),
    sqlc.arg(amount_in1),
    sqlc.arg(amount_out1),
    sqlc.arg(amount_in2),
    sqlc.arg(amount_out2),
    sqlc.arg(amount_in3),
    sqlc.arg(amount_out3),
    sqlc.arg(package_count),
    sqlc.narg(package_type_id),
    sqlc.arg(raw_json)
)
RETURNING *;

-- name: UpsertIikoPreparedChart :one
INSERT INTO iiko_prepared_charts (
    id,
    assembled_product_id,
    date_from,
    date_to,
    product_size_assembly_strategy,
    effective_direct_writeoff_store_spec_json,
    raw_json,
    updated_at
) VALUES (
    sqlc.arg(id),
    sqlc.arg(assembled_product_id),
    sqlc.arg(date_from),
    sqlc.narg(date_to),
    sqlc.arg(product_size_assembly_strategy),
    sqlc.arg(effective_direct_writeoff_store_spec_json),
    sqlc.arg(raw_json),
    sqlc.arg(updated_at)
)
ON CONFLICT(id) DO UPDATE SET
    assembled_product_id = excluded.assembled_product_id,
    date_from = excluded.date_from,
    date_to = excluded.date_to,
    product_size_assembly_strategy = excluded.product_size_assembly_strategy,
    effective_direct_writeoff_store_spec_json = excluded.effective_direct_writeoff_store_spec_json,
    raw_json = excluded.raw_json,
    updated_at = excluded.updated_at
RETURNING *;

-- name: DeleteIikoPreparedChartItemsByChartID :exec
DELETE FROM iiko_prepared_chart_items
WHERE prepared_chart_id = sqlc.arg(prepared_chart_id);

-- name: InsertIikoPreparedChartItem :one
INSERT INTO iiko_prepared_chart_items (
    id,
    prepared_chart_id,
    sort_weight,
    product_id,
    product_size_spec_json,
    store_spec_json,
    amount,
    raw_json
) VALUES (
    sqlc.arg(id),
    sqlc.arg(prepared_chart_id),
    sqlc.arg(sort_weight),
    sqlc.arg(product_id),
    sqlc.arg(product_size_spec_json),
    sqlc.arg(store_spec_json),
    sqlc.arg(amount),
    sqlc.arg(raw_json)
)
RETURNING *;
