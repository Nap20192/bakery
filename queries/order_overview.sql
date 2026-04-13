-- name: GetOrderIngredients :many
WITH RECURSIVE bom AS (
    SELECT
        oi.quantity AS multiplier,
        tci.ingredient_id,
        tci.sub_product_id,
        tci.net_qty::numeric AS net_qty,
        1 AS depth
    FROM order_items oi
    JOIN tech_cards tc ON tc.product_id = oi.product_id
    JOIN tech_card_items tci ON tci.tech_card_id = tc.id
    WHERE oi.order_id = $1

    UNION ALL

    SELECT
        bom.multiplier,
        tci.ingredient_id,
        tci.sub_product_id,
        (tci.net_qty * bom.net_qty)::numeric AS net_qty,
        bom.depth + 1
    FROM bom
    JOIN tech_cards tc ON tc.product_id = bom.sub_product_id
    JOIN tech_card_items tci ON tci.tech_card_id = tc.id
    WHERE bom.sub_product_id IS NOT NULL AND bom.depth < 10
)
SELECT
    i.id,
    i.name AS ingredient_name,
    i.unit,
    SUM(bom.net_qty * bom.multiplier)::float8 AS total_qty
FROM bom
JOIN ingredients i ON i.id = bom.ingredient_id
WHERE bom.ingredient_id IS NOT NULL
GROUP BY i.id, i.name, i.unit
ORDER BY i.name;
