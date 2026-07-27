-- +goose Up
-- Order drafts: a shop user's unfinished заявка, saved from the order-creation
-- form and reloaded later. One draft per (user, category) — saving again for
-- the same category overwrites the previous draft (UNIQUE below is what
-- enforces this; the app upserts on it). Items/comments are stored as JSONB
-- since a draft is a single provisional blob, not an aggregate with its own
-- item rows or events. Consumed (deleted) once the real order is created;
-- otherwise purged by the same cleanup ticker that prunes old orders.
CREATE TABLE IF NOT EXISTS order_drafts (
    id BIGSERIAL PRIMARY KEY,
    created_by_username TEXT NOT NULL,
    category_id BIGINT NOT NULL REFERENCES order_categories(id) ON DELETE CASCADE,
    from_department_id BIGINT REFERENCES departments(id) ON DELETE SET NULL,
    fulfillment_date DATE NOT NULL,
    comments JSONB,
    items JSONB NOT NULL DEFAULT '[]'::jsonb,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (created_by_username, category_id)
);

-- +goose Down
DROP TABLE IF EXISTS order_drafts;
