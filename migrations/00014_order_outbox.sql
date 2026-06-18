-- +goose Up
-- Transactional outbox for order domain events. Events are written in the same
-- transaction as the order change, then a relay publishes them to RabbitMQ and
-- stamps published_at. This guarantees at-least-once delivery even if the
-- process crashes between commit and publish.
CREATE TABLE IF NOT EXISTS order_outbox (
    id             BIGSERIAL PRIMARY KEY,
    aggregate_id   TEXT        NOT NULL,
    event_type     TEXT        NOT NULL,
    payload        JSONB       NOT NULL,
    correlation_id TEXT        NOT NULL DEFAULT '',
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    published_at   TIMESTAMPTZ
);

-- Partial index: the relay only ever scans unpublished rows.
CREATE INDEX IF NOT EXISTS idx_order_outbox_unpublished
ON order_outbox (id)
WHERE published_at IS NULL;

-- +goose Down
DROP TABLE IF EXISTS order_outbox;
