-- name: TechCardInsert :one
INSERT INTO tech_cards (product_id, version, source, source_ref, yield_qty, yield_unit, valid_from, valid_to)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING *;

-- name: TechCardGetActive :one
SELECT * FROM tech_cards
WHERE product_id = $1
  AND deleted_at IS NULL
  AND valid_from <= $2
  AND (valid_to IS NULL OR valid_to >= $2)
ORDER BY version DESC
LIMIT 1;

-- name: TechCardGetByID :one
SELECT * FROM tech_cards
WHERE id = $1 AND deleted_at IS NULL;

-- name: TechCardListByProduct :many
SELECT * FROM tech_cards
WHERE product_id = $1 AND deleted_at IS NULL
ORDER BY version DESC;

-- name: TechCardSoftDelete :exec
UPDATE tech_cards SET deleted_at = now()
WHERE id = $1 AND deleted_at IS NULL;

-- name: TechCardItemInsert :one
INSERT INTO tech_card_items (tech_card_id, ingredient_id, gross_qty, net_qty, unit)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (tech_card_id, ingredient_id) DO UPDATE
    SET gross_qty = EXCLUDED.gross_qty,
        net_qty = EXCLUDED.net_qty,
        unit = EXCLUDED.unit,
        updated_at = now()
RETURNING *;

-- name: TechCardItemListByCard :many
SELECT tci.*, i.name AS ingredient_name, i.is_dough
FROM tech_card_items tci
JOIN ingredients i ON i.id = tci.ingredient_id
WHERE tci.tech_card_id = $1
ORDER BY i.name;

-- name: TechCardItemListDoughByCard :many
SELECT tci.*, i.name AS ingredient_name
FROM tech_card_items tci
JOIN ingredients i ON i.id = tci.ingredient_id
WHERE tci.tech_card_id = $1
  AND i.is_dough = true
  AND i.deleted_at IS NULL
ORDER BY i.name;

-- name: TechCardItemDeleteByCard :exec
DELETE FROM tech_card_items WHERE tech_card_id = $1;
