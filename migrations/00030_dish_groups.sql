-- +goose Up
-- Normalize the dish "группа" (was a free-text `theme` string repeated on every
-- dish) into its own table. A group belongs to a category; a dish belongs to a
-- category and a group. Three normalized tables: order_categories → dish_groups
-- → dish_catalog.
CREATE TABLE IF NOT EXISTS dish_groups (
    id BIGSERIAL PRIMARY KEY,
    category_id BIGINT REFERENCES order_categories(id) ON DELETE SET NULL,
    name TEXT NOT NULL,
    sort_order BIGINT NOT NULL DEFAULT 0,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (category_id, name)
);

-- Backfill one group per distinct (category, theme) from the existing catalog.
INSERT INTO dish_groups (category_id, name, sort_order)
SELECT category_id, theme, MIN(sort_order)
FROM dish_catalog
WHERE trim(theme) <> ''
GROUP BY category_id, theme;

ALTER TABLE dish_catalog
ADD COLUMN IF NOT EXISTS group_id BIGINT REFERENCES dish_groups(id) ON DELETE SET NULL;

UPDATE dish_catalog d
SET group_id = g.id
FROM dish_groups g
WHERE g.name = d.theme
  AND g.category_id IS NOT DISTINCT FROM d.category_id;

CREATE INDEX IF NOT EXISTS idx_dish_catalog_group_id ON dish_catalog(group_id);

ALTER TABLE dish_catalog DROP COLUMN IF EXISTS theme;

-- +goose Down
ALTER TABLE dish_catalog ADD COLUMN IF NOT EXISTS theme TEXT NOT NULL DEFAULT '';

UPDATE dish_catalog d
SET theme = g.name
FROM dish_groups g
WHERE g.id = d.group_id;

DROP INDEX IF EXISTS idx_dish_catalog_group_id;
ALTER TABLE dish_catalog DROP COLUMN IF EXISTS group_id;
DROP TABLE IF EXISTS dish_groups;
