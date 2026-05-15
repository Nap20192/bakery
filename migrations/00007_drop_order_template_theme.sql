-- +goose Up
DROP INDEX IF EXISTS idx_order_templates_theme;

ALTER TABLE order_templates
DROP COLUMN IF EXISTS theme;

-- +goose Down
ALTER TABLE order_templates
ADD COLUMN IF NOT EXISTS theme TEXT NOT NULL DEFAULT '';

CREATE INDEX IF NOT EXISTS idx_order_templates_theme
ON order_templates(theme);
