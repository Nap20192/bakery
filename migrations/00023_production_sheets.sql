-- +goose Up
-- Журнал отработок: каждая отработка — отдельный документ, который можно
-- открыть, изменить и удалить. Факт в позициях заказов
-- (order_items.produced_quantity) — денормализованная проекция журнала,
-- пересчитывается при любом изменении листа (последний лист побеждает).
CREATE TABLE IF NOT EXISTS production_sheets (
    id BIGSERIAL PRIMARY KEY,
    created_by_username TEXT NOT NULL DEFAULT '',
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS production_sheet_items (
    id BIGSERIAL PRIMARY KEY,
    sheet_id BIGINT NOT NULL REFERENCES production_sheets(id) ON DELETE CASCADE,
    order_id BIGINT NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    product_name TEXT NOT NULL,
    produced_quantity DOUBLE PRECISION NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_production_sheet_items_sheet ON production_sheet_items(sheet_id);
CREATE INDEX IF NOT EXISTS idx_production_sheet_items_order ON production_sheet_items(order_id);

-- +goose Down
DROP TABLE IF EXISTS production_sheet_items;
DROP TABLE IF EXISTS production_sheets;
