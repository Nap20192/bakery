-- +goose Up
-- Retention cleanup deletes by created_at; index the non-favorite rows it scans.
CREATE INDEX IF NOT EXISTS idx_orders_created_at_not_favorite
    ON orders (created_at)
    WHERE is_favorite = FALSE;

-- +goose Down
DROP INDEX IF EXISTS idx_orders_created_at_not_favorite;
