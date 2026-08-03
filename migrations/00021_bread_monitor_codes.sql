-- +goose Up
-- Dough codes for the «Хлеб» category (found by walking the iiko tech cards
-- of every bread dish; see templates/monitor_codes.txt for the breakdown).
-- Only fills the list when it is still empty, so admin edits are never
-- overwritten.
UPDATE order_categories
SET monitor_codes = ARRAY['21429', '17642', '21212', '21192'], updated_at = now()
WHERE code = 'bread' AND monitor_codes = '{}';

-- +goose Down
UPDATE order_categories
SET monitor_codes = '{}', updated_at = now()
WHERE code = 'bread' AND monitor_codes = ARRAY['21429', '17642', '21212', '21192'];
