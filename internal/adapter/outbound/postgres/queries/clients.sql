-- name: ClientInsert :one
INSERT INTO clients (name, telegram_chat_id, phone, note)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: ClientGetByID :one
SELECT * FROM clients
WHERE id = $1 AND deleted_at IS NULL;

-- name: ClientGetByTelegramChatID :one
SELECT * FROM clients
WHERE telegram_chat_id = $1 AND deleted_at IS NULL;

-- name: ClientList :many
SELECT * FROM clients
WHERE deleted_at IS NULL
ORDER BY name;

-- name: ClientUpdate :one
UPDATE clients
SET name = $2, phone = $3, note = $4
WHERE id = $1 AND deleted_at IS NULL
RETURNING *;

-- name: ClientSoftDelete :exec
UPDATE clients SET deleted_at = now()
WHERE id = $1 AND deleted_at IS NULL;
