-- +goose Up
ALTER TABLE orders
ADD COLUMN IF NOT EXISTS created_by_username TEXT NOT NULL DEFAULT '';

CREATE INDEX IF NOT EXISTS idx_orders_created_by_username
ON orders(created_by_username)
WHERE created_by_username <> '';

-- +goose Down
DROP INDEX IF EXISTS idx_orders_created_by_username;

ALTER TABLE orders
DROP COLUMN IF EXISTS created_by_username;
