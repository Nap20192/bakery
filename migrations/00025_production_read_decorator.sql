-- +goose Up
-- Отработка становится read-time декоратором: журнал (production_sheets +
-- production_sheet_items) — единственное место, где живёт факт выпечки.
-- Заказ больше не мутируется: колонки-проекции produced_* удаляются (факт
-- присоединяется к позициям заказа при чтении), записи истории «produced»
-- удаляются — отработка не является изменением заказа.
ALTER TABLE order_items DROP COLUMN IF EXISTS produced_quantity;
ALTER TABLE order_items DROP COLUMN IF EXISTS produced_reason;

DELETE FROM order_history_items WHERE change_type = 'produced';
DELETE FROM order_history h
WHERE NOT EXISTS (
    SELECT 1 FROM order_history_items i WHERE i.history_id = h.id
);

ALTER TABLE order_history_items DROP CONSTRAINT IF EXISTS order_history_items_change_type_check;
ALTER TABLE order_history_items ADD CONSTRAINT order_history_items_change_type_check
CHECK (change_type = ANY (ARRAY['added'::text, 'updated'::text, 'removed'::text]));

-- Декорирующий JOIN ищет по (order_id, имя позиции) — индекс на журнал.
CREATE INDEX IF NOT EXISTS production_sheet_items_order_product_idx
ON production_sheet_items (order_id, lower(trim(product_name)), sheet_id DESC);

-- +goose Down
DROP INDEX IF EXISTS production_sheet_items_order_product_idx;

ALTER TABLE order_history_items DROP CONSTRAINT IF EXISTS order_history_items_change_type_check;
ALTER TABLE order_history_items ADD CONSTRAINT order_history_items_change_type_check
CHECK (change_type = ANY (ARRAY['added'::text, 'updated'::text, 'removed'::text, 'produced'::text]));

ALTER TABLE order_items ADD COLUMN IF NOT EXISTS produced_quantity DOUBLE PRECISION;
ALTER TABLE order_items ADD COLUMN IF NOT EXISTS produced_reason TEXT;
