-- +goose Up
-- Обоснование отклонения отработки («подгорело», «упало», …): хранится на
-- строке документа и проецируется в позицию заказа вместе с фактом, чтобы
-- магазин видел причину недовоза/излишка.
ALTER TABLE production_sheet_items
ADD COLUMN IF NOT EXISTS reason TEXT NOT NULL DEFAULT '';

ALTER TABLE order_items
ADD COLUMN IF NOT EXISTS produced_reason TEXT;

-- +goose Down
ALTER TABLE order_items DROP COLUMN IF EXISTS produced_reason;
ALTER TABLE production_sheet_items DROP COLUMN IF EXISTS reason;
