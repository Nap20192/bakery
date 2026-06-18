-- name: InsertOrderOutboxEvent :one
INSERT INTO order_outbox (
    aggregate_id,
    event_type,
    payload,
    correlation_id
) VALUES (
    sqlc.arg(aggregate_id),
    sqlc.arg(event_type),
    sqlc.arg(payload),
    sqlc.arg(correlation_id)
)
RETURNING id;

-- name: ListUnpublishedOrderOutboxEvents :many
SELECT id, aggregate_id, event_type, payload, correlation_id, created_at
FROM order_outbox
WHERE published_at IS NULL
ORDER BY id
LIMIT sqlc.arg(max_events);

-- name: MarkOrderOutboxEventPublished :exec
UPDATE order_outbox
SET published_at = now()
WHERE id = sqlc.arg(id);
