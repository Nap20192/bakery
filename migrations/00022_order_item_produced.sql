-- +goose Up
-- Отработка (факт выпечки): сколько реально испечено по позиции заказа.
-- NULL = отработка ещё не вносилась. Заявка (quantity/reserved_quantity)
-- не изменяется — «просили» и «испекли» живут рядом.
ALTER TABLE order_items
ADD COLUMN IF NOT EXISTS produced_quantity DOUBLE PRECISION;

-- История изменений получает новый тип записи «produced» (отработка).
ALTER TABLE order_history_items DROP CONSTRAINT IF EXISTS order_history_items_change_type_check;
ALTER TABLE order_history_items ADD CONSTRAINT order_history_items_change_type_check
CHECK (change_type = ANY (ARRAY['added'::text, 'updated'::text, 'removed'::text, 'produced'::text]));

-- +goose Down
ALTER TABLE order_history_items DROP CONSTRAINT IF EXISTS order_history_items_change_type_check;
ALTER TABLE order_history_items ADD CONSTRAINT order_history_items_change_type_check
CHECK (change_type = ANY (ARRAY['added'::text, 'updated'::text, 'removed'::text]));

ALTER TABLE order_items DROP COLUMN IF EXISTS produced_quantity;
