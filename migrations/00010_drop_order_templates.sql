-- +goose Up
DROP TABLE IF EXISTS order_templates;

-- +goose Down
CREATE TABLE IF NOT EXISTS order_templates (
    id BIGSERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    body TEXT NOT NULL,
    created_by_user_id BIGINT REFERENCES auth_users(id) ON DELETE SET NULL,
    created_at TEXT NOT NULL DEFAULT (now()::text),
    updated_at TEXT NOT NULL DEFAULT (now()::text)
);

CREATE INDEX IF NOT EXISTS idx_order_templates_created_by_user_id
ON order_templates(created_by_user_id);
