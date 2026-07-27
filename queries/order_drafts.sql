-- name: UpsertOrderDraft :one
INSERT INTO order_drafts (
    created_by_username,
    category_id,
    from_department_id,
    fulfillment_date,
    comments,
    items,
    updated_at
) VALUES (
    sqlc.arg(created_by_username),
    sqlc.arg(category_id),
    sqlc.narg(from_department_id),
    sqlc.arg(fulfillment_date),
    sqlc.narg(comments),
    sqlc.arg(items),
    now()
)
ON CONFLICT (created_by_username, category_id) DO UPDATE SET
    from_department_id = EXCLUDED.from_department_id,
    fulfillment_date = EXCLUDED.fulfillment_date,
    comments = EXCLUDED.comments,
    items = EXCLUDED.items,
    updated_at = now()
RETURNING *;

-- name: GetOrderDraft :one
SELECT *
FROM order_drafts
WHERE created_by_username = sqlc.arg(created_by_username)
  AND category_id = sqlc.arg(category_id);

-- name: ListOrderDrafts :many
SELECT *
FROM order_drafts
WHERE created_by_username = sqlc.arg(created_by_username)
ORDER BY updated_at DESC;

-- name: DeleteOrderDraft :exec
DELETE FROM order_drafts
WHERE created_by_username = sqlc.arg(created_by_username)
  AND category_id = sqlc.arg(category_id);

-- name: DeleteOrderDraftsOlderThan :one
WITH deleted AS (
    DELETE FROM order_drafts
    WHERE updated_at < sqlc.arg(updated_at_before)
    RETURNING id
)
SELECT COUNT(*)::BIGINT
FROM deleted;
