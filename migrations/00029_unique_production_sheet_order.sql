-- +goose Up
-- Один заказ может принадлежать только одному листу отработки. Проверка в
-- приложении остаётся для понятной ошибки, а constraint закрывает гонку между
-- параллельными транзакциями.
DO $$
DECLARE
    duplicate_order_ids TEXT;
BEGIN
    SELECT string_agg(order_id::TEXT, ', ' ORDER BY order_id)
    INTO duplicate_order_ids
    FROM (
        SELECT order_id
        FROM production_sheet_orders
        GROUP BY order_id
        HAVING COUNT(*) > 1
        ORDER BY order_id
        LIMIT 20
    ) duplicates;

    IF duplicate_order_ids IS NOT NULL THEN
        RAISE EXCEPTION
            'Нельзя закрепить один лист на заказ: заказы с несколькими листами (первые 20): %',
            duplicate_order_ids
            USING HINT = 'Разберите каждое пересечение вручную в production_sheet_orders и повторите миграцию.';
    END IF;
END
$$;

ALTER TABLE production_sheet_orders
ADD CONSTRAINT production_sheet_orders_order_id_key UNIQUE (order_id);

DROP INDEX IF EXISTS production_sheet_orders_order_idx;

-- +goose Down
ALTER TABLE production_sheet_orders
DROP CONSTRAINT IF EXISTS production_sheet_orders_order_id_key;

CREATE INDEX IF NOT EXISTS production_sheet_orders_order_idx
ON production_sheet_orders (order_id);
