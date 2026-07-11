-- +goose Up
-- Лист отработки фиксирует выбранные заказы (партию) независимо от того,
-- есть ли по ним отклонения: production_sheet_orders — сохранённый выбор,
-- production_sheet_items — только отклонения. Внутри листа считается тесто
-- по всей партии (факт учитывается read-time декоратором).
CREATE TABLE IF NOT EXISTS production_sheet_orders (
    sheet_id BIGINT NOT NULL REFERENCES production_sheets(id) ON DELETE CASCADE,
    order_id BIGINT NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    PRIMARY KEY (sheet_id, order_id)
);

CREATE INDEX IF NOT EXISTS production_sheet_orders_order_idx
ON production_sheet_orders (order_id);

-- Бэкфилл: существующие листы покрывали заказы только через отклонения.
INSERT INTO production_sheet_orders (sheet_id, order_id)
SELECT DISTINCT sheet_id, order_id
FROM production_sheet_items
ON CONFLICT DO NOTHING;

-- +goose Down
DROP TABLE IF EXISTS production_sheet_orders;
