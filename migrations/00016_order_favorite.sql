-- +goose Up
-- Favorite orders are pinned by an admin and must survive the retention cleanup.
ALTER TABLE orders ADD COLUMN IF NOT EXISTS is_favorite BOOLEAN NOT NULL DEFAULT FALSE;

-- +goose Down
ALTER TABLE orders DROP COLUMN IF EXISTS is_favorite;
