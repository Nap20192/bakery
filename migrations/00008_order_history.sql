-- +goose Up
CREATE TABLE IF NOT EXISTS order_history (
    id BIGSERIAL PRIMARY KEY,
    order_id BIGINT NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    changed_by_username TEXT NOT NULL DEFAULT '',
    changed_at TEXT NOT NULL DEFAULT (now()::text)
);

CREATE TABLE IF NOT EXISTS order_history_items (
    id BIGSERIAL PRIMARY KEY,
    history_id BIGINT NOT NULL REFERENCES order_history(id) ON DELETE CASCADE,
    change_type TEXT NOT NULL CHECK (change_type IN ('added', 'updated', 'removed')),
    product_code TEXT NOT NULL DEFAULT '',
    product_name TEXT NOT NULL DEFAULT '',
    old_quantity DOUBLE PRECISION,
    new_quantity DOUBLE PRECISION,
    old_reserved_quantity DOUBLE PRECISION,
    new_reserved_quantity DOUBLE PRECISION
);

CREATE INDEX IF NOT EXISTS idx_order_history_order_id
ON order_history(order_id);

CREATE INDEX IF NOT EXISTS idx_order_history_changed_at
ON order_history(changed_at);

CREATE INDEX IF NOT EXISTS idx_order_history_items_history_id
ON order_history_items(history_id);

-- +goose Down
DROP INDEX IF EXISTS idx_order_history_items_history_id;
DROP INDEX IF EXISTS idx_order_history_changed_at;
DROP INDEX IF EXISTS idx_order_history_order_id;

DROP TABLE IF EXISTS order_history_items;
DROP TABLE IF EXISTS order_history;
