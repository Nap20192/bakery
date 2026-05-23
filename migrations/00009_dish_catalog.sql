-- +goose Up
CREATE TABLE IF NOT EXISTS dish_catalog (
    id BIGSERIAL PRIMARY KEY,
    code TEXT NOT NULL,
    name TEXT NOT NULL,
    theme TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL DEFAULT '',
    updated_at TEXT NOT NULL DEFAULT ''
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_dish_catalog_code
ON dish_catalog(code);

CREATE INDEX IF NOT EXISTS idx_dish_catalog_normalized_name
ON dish_catalog(lower(trim(name)));

-- +goose Down
DROP INDEX IF EXISTS idx_dish_catalog_normalized_name;
DROP INDEX IF EXISTS idx_dish_catalog_code;
DROP TABLE IF EXISTS dish_catalog;
