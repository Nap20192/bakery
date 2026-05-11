-- +goose Up
ALTER TABLE auth_users
ADD COLUMN IF NOT EXISTS telegram_username TEXT;

ALTER TABLE orders
ADD COLUMN IF NOT EXISTS from_department_id BIGINT REFERENCES departments(id) ON DELETE SET NULL,
ADD COLUMN IF NOT EXISTS to_department_id BIGINT REFERENCES departments(id) ON DELETE SET NULL;

INSERT INTO departments (type, code, name)
VALUES
    ('shop', 'gagarina', 'Магазин Гагарина'),
    ('shop', 'saryarka', 'Магазин Сарыарка'),
    ('shop', 'sholokhova', 'Магазин Шолохова'),
    ('workshop', 'pekari', 'Цех Пекари')
ON CONFLICT(code) WHERE code <> '' DO UPDATE SET
    type = excluded.type,
    name = excluded.name,
    updated_at = now()::text;

CREATE INDEX IF NOT EXISTS idx_orders_from_department_id
ON orders(from_department_id);

CREATE INDEX IF NOT EXISTS idx_orders_to_department_id
ON orders(to_department_id);

CREATE INDEX IF NOT EXISTS idx_auth_users_telegram_username
ON auth_users(telegram_username)
WHERE telegram_username IS NOT NULL AND telegram_username <> '';

-- +goose Down
DROP INDEX IF EXISTS idx_auth_users_telegram_username;
DROP INDEX IF EXISTS idx_orders_to_department_id;
DROP INDEX IF EXISTS idx_orders_from_department_id;

ALTER TABLE orders
DROP COLUMN IF EXISTS to_department_id,
DROP COLUMN IF EXISTS from_department_id;

ALTER TABLE auth_users
DROP COLUMN IF EXISTS telegram_username;
