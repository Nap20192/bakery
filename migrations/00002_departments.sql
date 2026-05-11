-- +goose Up
CREATE TABLE IF NOT EXISTS departments (
    id BIGSERIAL PRIMARY KEY,
    type TEXT NOT NULL CHECK (type IN ('shop', 'workshop')),
    code TEXT NOT NULL DEFAULT '',
    name TEXT NOT NULL,
    iiko_department_id TEXT UNIQUE,
    metadata_json TEXT NOT NULL DEFAULT '{}',
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TEXT NOT NULL DEFAULT (now()::text),
    updated_at TEXT NOT NULL DEFAULT (now()::text)
);

ALTER TABLE auth_users
ADD COLUMN IF NOT EXISTS department_id BIGINT REFERENCES departments(id) ON DELETE SET NULL;

CREATE UNIQUE INDEX IF NOT EXISTS idx_departments_code
ON departments(code)
WHERE code <> '';

CREATE INDEX IF NOT EXISTS idx_departments_type
ON departments(type);

CREATE INDEX IF NOT EXISTS idx_departments_active
ON departments(is_active);

CREATE INDEX IF NOT EXISTS idx_auth_users_department_id
ON auth_users(department_id);

-- +goose Down
DROP INDEX IF EXISTS idx_auth_users_department_id;
DROP INDEX IF EXISTS idx_departments_active;
DROP INDEX IF EXISTS idx_departments_type;
DROP INDEX IF EXISTS idx_departments_code;

ALTER TABLE auth_users
DROP COLUMN IF EXISTS department_id;

DROP TABLE IF EXISTS departments;
