-- name: CreateTechCard :one
INSERT INTO tech_cards (id, product_id, yield_qty, yield_unit)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: GetTechCardByID :one
SELECT * FROM tech_cards WHERE id = $1;

-- name: GetTechCardByProductID :one
SELECT * FROM tech_cards
WHERE product_id = $1
ORDER BY created_at DESC
LIMIT 1;

-- name: ListTechCards :many
SELECT * FROM tech_cards ORDER BY created_at DESC;

-- name: ListTechCardsByProductID :many
SELECT * FROM tech_cards
WHERE product_id = $1
ORDER BY created_at DESC;

-- name: UpdateTechCard :one
UPDATE tech_cards
SET yield_qty = $2,
    yield_unit = $3,
    updated_at = now()
WHERE id = $1
RETURNING *;

-- name: DeleteTechCard :exec
DELETE FROM tech_cards WHERE id = $1;

-- name: CreateTechCardItem :one
INSERT INTO tech_card_items (
    id, tech_card_id, ingredient_id, sub_product_id, gross_qty, net_qty, unit
) VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: ListTechCardItems :many
SELECT * FROM tech_card_items
WHERE tech_card_id = $1;

-- name: DeleteTechCardItems :exec
DELETE FROM tech_card_items WHERE tech_card_id = $1;
