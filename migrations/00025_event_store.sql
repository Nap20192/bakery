-- +goose Up
-- Event store для event-sourced агрегатов (github.com/hallgren/eventsourcing,
-- eventstore/sql). Схема — ровно та, которую ожидает библиотека: поток
-- событий и есть источник истины; orders/order_items остаются read model.
CREATE TABLE IF NOT EXISTS events (
    seq SERIAL PRIMARY KEY,
    id VARCHAR NOT NULL,
    version INTEGER,
    reason VARCHAR,
    type VARCHAR,
    timestamp VARCHAR,
    data BYTEA,
    metadata BYTEA,
    UNIQUE (id, type, version)
);

CREATE INDEX IF NOT EXISTS id_type ON events (id, type);

-- +goose Down
DROP TABLE IF EXISTS events;
