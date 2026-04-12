-- name: CreateClient :one
INSERT INTO clients (id, user_id, name, telegram_chat_id, note)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: GetClientByID :one
SELECT * FROM clients WHERE id = $1;

-- name: GetClientByUserID :one
SELECT * FROM clients WHERE user_id = $1;

-- name: GetClientByTelegramChatID :one
SELECT * FROM clients WHERE telegram_chat_id = $1;

-- name: ListClients :many
SELECT * FROM clients ORDER BY name;

-- name: UpdateClient :one
UPDATE clients
SET name = $2,
    telegram_chat_id = $3,
    note = $4,
    updated_at = now()
WHERE id = $1
RETURNING *;

-- name: DeleteClient :exec
DELETE FROM clients WHERE id = $1;
