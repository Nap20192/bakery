-- +goose Up
-- Order categories (типы заявки): dynamic groups like «Хлеб» / «Булочки».
-- Each category carries a letter (used in the order number) and a color slug
-- (used by the frontend to highlight its orders).
CREATE TABLE IF NOT EXISTS order_categories (
    id BIGSERIAL PRIMARY KEY,
    code TEXT NOT NULL UNIQUE,
    letter TEXT NOT NULL DEFAULT '',
    name TEXT NOT NULL,
    color TEXT NOT NULL DEFAULT 'stone',
    sort_order BIGINT NOT NULL DEFAULT 0,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

INSERT INTO order_categories (code, letter, name, color, sort_order)
VALUES
    ('bread', 'Х', 'Хлеб', 'amber', 1),
    ('buns', 'Б', 'Булочки', 'sky', 2)
ON CONFLICT (code) DO NOTHING;

-- Orders belong to a category. Existing orders are backfilled to «Булочки»
-- (the current assortment is all pastry). ON DELETE SET NULL keeps orders
-- intact if an admin removes a category.
ALTER TABLE orders
ADD COLUMN IF NOT EXISTS category_id BIGINT REFERENCES order_categories(id) ON DELETE SET NULL;

UPDATE orders
SET category_id = (SELECT id FROM order_categories WHERE code = 'buns')
WHERE category_id IS NULL;

CREATE INDEX IF NOT EXISTS idx_orders_category_id ON orders(category_id);

-- Dish catalog entries belong to a category: the shop sees only the picked
-- category's dishes when composing an order.
ALTER TABLE dish_catalog
ADD COLUMN IF NOT EXISTS category_id BIGINT REFERENCES order_categories(id) ON DELETE SET NULL;

UPDATE dish_catalog
SET category_id = (SELECT id FROM order_categories WHERE code = 'buns')
WHERE category_id IS NULL;

-- Order numbers are sequential per shop per category per day, so each
-- category gets its own clean sequence (Г.Х…001, Г.Б…001).
ALTER TABLE order_counters
ADD COLUMN IF NOT EXISTS category_id BIGINT NOT NULL DEFAULT 0;

ALTER TABLE order_counters DROP CONSTRAINT IF EXISTS order_counters_pkey;
ALTER TABLE order_counters ADD PRIMARY KEY (day, department_id, category_id);

-- +goose Down
ALTER TABLE order_counters DROP CONSTRAINT IF EXISTS order_counters_pkey;
ALTER TABLE order_counters DROP COLUMN IF EXISTS category_id;
ALTER TABLE order_counters ADD PRIMARY KEY (day, department_id);

ALTER TABLE dish_catalog DROP COLUMN IF EXISTS category_id;

DROP INDEX IF EXISTS idx_orders_category_id;
ALTER TABLE orders DROP COLUMN IF EXISTS category_id;

DROP TABLE IF EXISTS order_categories;
