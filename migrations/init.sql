CREATE TABLE IF NOT EXISTS order_counters (
    day date PRIMARY KEY,
    counter INT NOT NULL
);

CREATE TABLE IF NOT EXISTS orders (
    id SERIAL PRIMARY KEY,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
;

CREATE TABLE IF NOT EXISTS order_items (
    id SERIAL PRIMARY KEY,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    order_id INTEGER REFERENCES orders (id),
    name TEXT NOT NULL,
    quantity INTEGER NOT NULL
);
