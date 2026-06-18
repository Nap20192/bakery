-- +goose Up
-- Per-order comments: a general note plus optional per-item notes, stored as
-- JSON. Items are keyed by product_name (stable and known to the client at
-- creation time, unlike row ids which are reassigned when items are rewritten).
-- Shape: {"general": "...", "items": [{"product_name": "...", "comment": "..."}]}
ALTER TABLE orders ADD COLUMN IF NOT EXISTS comments JSONB;

-- +goose Down
ALTER TABLE orders DROP COLUMN IF EXISTS comments;
