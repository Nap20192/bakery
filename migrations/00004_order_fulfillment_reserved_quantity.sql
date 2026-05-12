-- +goose Up
ALTER TABLE orders
ADD COLUMN IF NOT EXISTS fulfillment_date TEXT NOT NULL DEFAULT (CURRENT_DATE::text);

ALTER TABLE order_items
ADD COLUMN IF NOT EXISTS reserved_quantity DOUBLE PRECISION NOT NULL DEFAULT 0;

CREATE INDEX IF NOT EXISTS idx_orders_fulfillment_date
ON orders(fulfillment_date);

-- +goose Down
DROP INDEX IF EXISTS idx_orders_fulfillment_date;

ALTER TABLE order_items
DROP COLUMN IF EXISTS reserved_quantity;

ALTER TABLE orders
DROP COLUMN IF EXISTS fulfillment_date;
