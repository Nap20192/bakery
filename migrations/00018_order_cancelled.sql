-- +goose Up
-- Orders can be cancelled (soft state): a non-null cancelled_at marks the order
-- as cancelled. Cancelled orders cannot be edited until restored.
ALTER TABLE orders ADD COLUMN IF NOT EXISTS cancelled_at TIMESTAMPTZ;
ALTER TABLE orders ADD COLUMN IF NOT EXISTS cancelled_by_username TEXT NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE orders DROP COLUMN IF EXISTS cancelled_by_username;
ALTER TABLE orders DROP COLUMN IF EXISTS cancelled_at;
