-- +goose Up
-- dedup_key — детерминированный отпечаток заявки (автор, источник, категория,
-- дата выполнения, день создания, состав). Частичный UNIQUE-индекс закрывает
-- гонку двойного создания (быстрый двойной клик, ретрай сети, повторный POST из
-- Telegram-webview): вторая идентичная заявка за день не пройдёт. Индекс
-- частичный — отменённые заявки не блокируют повторное создание, а строки без
-- ключа (созданные до миграции) в проверке не участвуют.
ALTER TABLE orders ADD COLUMN IF NOT EXISTS dedup_key TEXT;

CREATE UNIQUE INDEX IF NOT EXISTS orders_dedup_key_active_uniq
ON orders (dedup_key)
WHERE dedup_key IS NOT NULL AND cancelled_at IS NULL;

-- +goose Down
DROP INDEX IF EXISTS orders_dedup_key_active_uniq;

ALTER TABLE orders DROP COLUMN IF EXISTS dedup_key;
