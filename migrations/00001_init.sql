-- +goose Up
PRAGMA foreign_keys = ON;

CREATE TABLE IF NOT EXISTS groups (
    code TEXT PRIMARY KEY,
    name TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS chats_metadata(
    chat_id string PRIMARY KEY,
    metadata_json TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS auth_users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    telegram_id INTEGER UNIQUE,
    username TEXT NOT NULL DEFAULT '',
    password_hash TEXT NOT NULL DEFAULT '',
    metadata_json TEXT NOT NULL DEFAULT '{}',
    role TEXT NOT NULL DEFAULT 'user',
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS order_counters (
    day TEXT PRIMARY KEY,
    counter INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS orders (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    number TEXT NOT NULL UNIQUE,
    location TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS order_items (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    order_id INTEGER NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    iiko_product_id TEXT REFERENCES iiko_products(id) ON DELETE SET NULL,
    product_name TEXT NOT NULL,
    quantity INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS iiko_sync_runs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    source TEXT NOT NULL,
    date_from TEXT NOT NULL DEFAULT '',
    date_to TEXT NOT NULL DEFAULT '',
    known_revision INTEGER NOT NULL DEFAULT -1,
    status TEXT NOT NULL,
    error TEXT NOT NULL DEFAULT '',
    started_at TEXT NOT NULL DEFAULT (datetime('now')),
    finished_at TEXT
);

CREATE TABLE IF NOT EXISTS iiko_products (
    id TEXT PRIMARY KEY,
    code TEXT NOT NULL DEFAULT '',
    name TEXT NOT NULL,
    type TEXT,
    measure_unit TEXT NOT NULL DEFAULT '',
    raw_json TEXT NOT NULL,
    updated_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS iiko_assembly_charts (
    id TEXT PRIMARY KEY,
    assembled_product_id TEXT NOT NULL,
    date_from TEXT NOT NULL,
    date_to TEXT,
    assembled_amount REAL NOT NULL DEFAULT 0,
    product_writeoff_strategy TEXT NOT NULL DEFAULT '',
    product_size_assembly_strategy TEXT NOT NULL DEFAULT '',
    effective_direct_writeoff_store_spec_json TEXT NOT NULL DEFAULT '',
    raw_json TEXT NOT NULL,
    updated_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS iiko_assembly_chart_items (
    id TEXT PRIMARY KEY,
    chart_id TEXT NOT NULL REFERENCES iiko_assembly_charts(id) ON DELETE CASCADE,
    sort_weight REAL NOT NULL DEFAULT 0,
    product_id TEXT NOT NULL,
    product_size_spec_json TEXT NOT NULL DEFAULT '',
    store_spec_json TEXT NOT NULL DEFAULT '',
    amount_in REAL NOT NULL DEFAULT 0,
    amount_middle REAL NOT NULL DEFAULT 0,
    amount_out REAL NOT NULL DEFAULT 0,
    amount_in1 REAL NOT NULL DEFAULT 0,
    amount_out1 REAL NOT NULL DEFAULT 0,
    amount_in2 REAL NOT NULL DEFAULT 0,
    amount_out2 REAL NOT NULL DEFAULT 0,
    amount_in3 REAL NOT NULL DEFAULT 0,
    amount_out3 REAL NOT NULL DEFAULT 0,
    package_count REAL NOT NULL DEFAULT 0,
    package_type_id TEXT,
    raw_json TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS iiko_prepared_charts (
    id TEXT PRIMARY KEY,
    assembled_product_id TEXT NOT NULL,
    date_from TEXT NOT NULL,
    date_to TEXT,
    product_size_assembly_strategy TEXT NOT NULL DEFAULT '',
    effective_direct_writeoff_store_spec_json TEXT NOT NULL DEFAULT '',
    raw_json TEXT NOT NULL,
    updated_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS iiko_prepared_chart_items (
    id TEXT PRIMARY KEY,
    prepared_chart_id TEXT NOT NULL REFERENCES iiko_prepared_charts(id) ON DELETE CASCADE,
    sort_weight REAL NOT NULL DEFAULT 0,
    product_id TEXT NOT NULL,
    product_size_spec_json TEXT NOT NULL DEFAULT '',
    store_spec_json TEXT NOT NULL DEFAULT '',
    amount REAL NOT NULL DEFAULT 0,
    raw_json TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_order_items_order_id ON order_items(order_id);
CREATE INDEX IF NOT EXISTS idx_order_items_iiko_product_id ON order_items(iiko_product_id);
CREATE INDEX IF NOT EXISTS idx_auth_users_role ON auth_users(role);
CREATE UNIQUE INDEX IF NOT EXISTS idx_auth_users_username ON auth_users(username) WHERE username <> '';
CREATE INDEX IF NOT EXISTS idx_iiko_products_name ON iiko_products(name);
CREATE INDEX IF NOT EXISTS idx_iiko_products_type ON iiko_products(type);
CREATE INDEX IF NOT EXISTS idx_iiko_assembly_charts_product ON iiko_assembly_charts(assembled_product_id);
CREATE INDEX IF NOT EXISTS idx_iiko_assembly_charts_period ON iiko_assembly_charts(date_from, date_to);
CREATE INDEX IF NOT EXISTS idx_iiko_assembly_chart_items_chart ON iiko_assembly_chart_items(chart_id);
CREATE INDEX IF NOT EXISTS idx_iiko_assembly_chart_items_product ON iiko_assembly_chart_items(product_id);
CREATE INDEX IF NOT EXISTS idx_iiko_prepared_charts_product ON iiko_prepared_charts(assembled_product_id);
CREATE INDEX IF NOT EXISTS idx_iiko_prepared_charts_period ON iiko_prepared_charts(date_from, date_to);
CREATE INDEX IF NOT EXISTS idx_iiko_prepared_chart_items_chart ON iiko_prepared_chart_items(prepared_chart_id);
CREATE INDEX IF NOT EXISTS idx_iiko_prepared_chart_items_product ON iiko_prepared_chart_items(product_id);

-- +goose Down
PRAGMA foreign_keys = OFF;

DROP INDEX IF EXISTS idx_iiko_prepared_chart_items_product;
DROP INDEX IF EXISTS idx_iiko_prepared_chart_items_chart;
DROP INDEX IF EXISTS idx_iiko_prepared_charts_period;
DROP INDEX IF EXISTS idx_iiko_prepared_charts_product;
DROP INDEX IF EXISTS idx_iiko_assembly_chart_items_product;
DROP INDEX IF EXISTS idx_iiko_assembly_chart_items_chart;
DROP INDEX IF EXISTS idx_iiko_assembly_charts_period;
DROP INDEX IF EXISTS idx_iiko_assembly_charts_product;
DROP INDEX IF EXISTS idx_iiko_products_type;
DROP INDEX IF EXISTS idx_iiko_products_name;
DROP INDEX IF EXISTS idx_auth_users_role;
DROP INDEX IF EXISTS idx_auth_users_username;
DROP INDEX IF EXISTS idx_order_items_iiko_product_id;
DROP INDEX IF EXISTS idx_order_items_order_id;

DROP TABLE IF EXISTS iiko_prepared_chart_items;
DROP TABLE IF EXISTS iiko_prepared_charts;
DROP TABLE IF EXISTS iiko_assembly_chart_items;
DROP TABLE IF EXISTS iiko_assembly_charts;
DROP TABLE IF EXISTS iiko_products;
DROP TABLE IF EXISTS iiko_sync_runs;
DROP TABLE IF EXISTS order_items;
DROP TABLE IF EXISTS orders;
DROP TABLE IF EXISTS order_counters;
DROP TABLE IF EXISTS auth_users;

PRAGMA foreign_keys = ON;
