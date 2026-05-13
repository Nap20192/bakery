-- +goose Up
CREATE TABLE IF NOT EXISTS order_templates (
    id BIGSERIAL PRIMARY KEY,
    theme TEXT NOT NULL,
    name TEXT NOT NULL,
    body TEXT NOT NULL,
    created_by_user_id BIGINT REFERENCES auth_users(id) ON DELETE SET NULL,
    created_at TEXT NOT NULL DEFAULT (now()::text),
    updated_at TEXT NOT NULL DEFAULT (now()::text)
);

CREATE INDEX IF NOT EXISTS idx_order_templates_theme
ON order_templates(theme);

CREATE INDEX IF NOT EXISTS idx_order_templates_created_by_user_id
ON order_templates(created_by_user_id);

-- +goose Down
DROP INDEX IF EXISTS idx_order_templates_created_by_user_id;
DROP INDEX IF EXISTS idx_order_templates_theme;

DROP TABLE IF EXISTS order_templates;
