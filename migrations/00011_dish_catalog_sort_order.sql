-- +goose Up
ALTER TABLE dish_catalog
ADD COLUMN IF NOT EXISTS sort_order BIGINT NOT NULL DEFAULT 0;

CREATE INDEX IF NOT EXISTS idx_dish_catalog_sort_order
ON dish_catalog(sort_order, id);

-- +goose Down
DROP INDEX IF EXISTS idx_dish_catalog_sort_order;

ALTER TABLE dish_catalog
DROP COLUMN IF EXISTS sort_order;
