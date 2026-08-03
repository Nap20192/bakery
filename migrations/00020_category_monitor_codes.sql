-- +goose Up
-- Dough codes for the monitor calculation are configured per тип заявки:
-- «Булочки» keep the previous hardcoded set, «Хлеб» starts empty (the admin
-- fills it in the admin panel).
ALTER TABLE order_categories
ADD COLUMN IF NOT EXISTS monitor_codes TEXT[] NOT NULL DEFAULT '{}';

UPDATE order_categories
SET monitor_codes = ARRAY['17642', '17644', '17650', '19694']
WHERE code = 'buns' AND monitor_codes = '{}';

-- +goose Down
ALTER TABLE order_categories DROP COLUMN IF EXISTS monitor_codes;
