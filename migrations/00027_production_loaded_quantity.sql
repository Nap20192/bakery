-- +goose Up
CREATE TABLE production_sheet_loads (
    sheet_id BIGINT NOT NULL REFERENCES production_sheets(id) ON DELETE CASCADE,
    order_id BIGINT NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    product_name TEXT NOT NULL,
    loaded_quantity DOUBLE PRECISION NOT NULL CHECK (loaded_quantity >= 0),
    PRIMARY KEY (sheet_id, order_id, product_name)
);

CREATE INDEX production_sheet_loads_order_id_idx
    ON production_sheet_loads (order_id);

-- Existing sheets predate the explicit «Закладка» field. Backfill every
-- selected order item with its requested production quantity.
INSERT INTO production_sheet_loads (sheet_id, order_id, product_name, loaded_quantity)
SELECT
    pso.sheet_id,
    pso.order_id,
    oi.product_name,
    oi.quantity + oi.reserved_quantity
FROM production_sheet_orders pso
JOIN order_items oi ON oi.order_id = pso.order_id
ON CONFLICT DO NOTHING;

-- +goose Down
DROP TABLE production_sheet_loads;
