-- +goose Up
CREATE TABLE departments (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    type TEXT NOT NULL CHECK (type IN ('shop', 'workshop')),
    code TEXT NOT NULL DEFAULT '',
    name TEXT NOT NULL,
    iiko_department_id TEXT UNIQUE,
    metadata_json TEXT NOT NULL DEFAULT '{}',
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at DATETIME NOT NULL DEFAULT (strftime('%Y-%m-%d %H:%M:%f+00:00', 'now')),
    updated_at DATETIME NOT NULL DEFAULT (strftime('%Y-%m-%d %H:%M:%f+00:00', 'now'))
);
CREATE UNIQUE INDEX idx_departments_code ON departments(code) WHERE code <> '';
CREATE INDEX idx_departments_type ON departments(type);
CREATE INDEX idx_departments_active ON departments(is_active);

INSERT INTO departments (type, code, name) VALUES
    ('shop', 'gagarina', 'Магазин Гагарина'),
    ('shop', 'saryarka', 'Магазин Сарыарка'),
    ('shop', 'sholokhova', 'Магазин Шолохова'),
    ('workshop', 'pekari', 'Цех Пекари');

CREATE TABLE auth_users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    telegram_id INTEGER UNIQUE,
    username TEXT NOT NULL DEFAULT '',
    password_hash TEXT NOT NULL DEFAULT '',
    metadata_json TEXT NOT NULL DEFAULT '{}',
    role TEXT NOT NULL DEFAULT 'user',
    created_at DATETIME NOT NULL DEFAULT (strftime('%Y-%m-%d %H:%M:%f+00:00', 'now')),
    updated_at DATETIME NOT NULL DEFAULT (strftime('%Y-%m-%d %H:%M:%f+00:00', 'now')),
    department_id INTEGER REFERENCES departments(id) ON DELETE SET NULL,
    telegram_username TEXT
);
CREATE INDEX idx_auth_users_role ON auth_users(role);
CREATE UNIQUE INDEX idx_auth_users_username ON auth_users(username) WHERE username <> '';
CREATE INDEX idx_auth_users_department_id ON auth_users(department_id);
CREATE UNIQUE INDEX idx_auth_users_telegram_username
    ON auth_users(telegram_username)
    WHERE telegram_username IS NOT NULL AND telegram_username <> '';

CREATE TABLE order_categories (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    code TEXT NOT NULL UNIQUE,
    letter TEXT NOT NULL DEFAULT '',
    name TEXT NOT NULL,
    color TEXT NOT NULL DEFAULT 'stone',
    sort_order INTEGER NOT NULL DEFAULT 0,
    created_at DATETIME NOT NULL DEFAULT (strftime('%Y-%m-%d %H:%M:%f+00:00', 'now')),
    updated_at DATETIME NOT NULL DEFAULT (strftime('%Y-%m-%d %H:%M:%f+00:00', 'now')),
    monitor_codes TEXT NOT NULL DEFAULT '[]'
);

INSERT INTO order_categories (code, letter, name, color, sort_order, monitor_codes) VALUES
    ('bread', 'Х', 'Хлеб', 'amber', 1, '["21429","17642","21212","21192"]'),
    ('buns', 'Б', 'Булочки', 'sky', 2, '["17642","17644","17650","19694"]');

CREATE TABLE order_counters (
    day TEXT NOT NULL,
    counter INTEGER NOT NULL DEFAULT 0,
    department_id INTEGER NOT NULL DEFAULT 0,
    category_id INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (day, department_id, category_id)
);

CREATE TABLE orders (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    number TEXT NOT NULL UNIQUE,
    location TEXT NOT NULL DEFAULT '',
    created_at DATETIME NOT NULL DEFAULT (strftime('%Y-%m-%d %H:%M:%f+00:00', 'now')),
    from_department_id INTEGER REFERENCES departments(id) ON DELETE RESTRICT,
    to_department_id INTEGER REFERENCES departments(id) ON DELETE RESTRICT,
    fulfillment_date TEXT NOT NULL CHECK (fulfillment_date = date(fulfillment_date)),
    created_by_username TEXT NOT NULL DEFAULT '',
    comments TEXT,
    is_favorite BOOLEAN NOT NULL DEFAULT FALSE,
    cancelled_at DATETIME,
    cancelled_by_username TEXT NOT NULL DEFAULT '',
    category_id INTEGER REFERENCES order_categories(id) ON DELETE SET NULL,
    dedup_key TEXT
);
CREATE INDEX idx_orders_from_department_id ON orders(from_department_id);
CREATE INDEX idx_orders_to_department_id ON orders(to_department_id);
CREATE INDEX idx_orders_fulfillment_date ON orders(fulfillment_date);
CREATE INDEX idx_orders_created_by_username ON orders(created_by_username) WHERE created_by_username <> '';
CREATE INDEX idx_orders_created_at_not_favorite ON orders(created_at) WHERE is_favorite = FALSE;
CREATE INDEX idx_orders_category_id ON orders(category_id);
CREATE UNIQUE INDEX orders_dedup_key_active_uniq
    ON orders(dedup_key)
    WHERE dedup_key IS NOT NULL AND cancelled_at IS NULL;

CREATE TABLE iiko_products (
    id TEXT PRIMARY KEY,
    code TEXT NOT NULL DEFAULT '',
    name TEXT NOT NULL,
    type TEXT,
    measure_unit TEXT NOT NULL DEFAULT '',
    raw_json TEXT NOT NULL,
    updated_at DATETIME NOT NULL DEFAULT (strftime('%Y-%m-%d %H:%M:%f+00:00', 'now'))
);
CREATE INDEX idx_iiko_products_name ON iiko_products(name);
CREATE INDEX idx_iiko_products_code ON iiko_products(code);
CREATE INDEX idx_iiko_products_type ON iiko_products(type);

CREATE TABLE order_items (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    order_id INTEGER NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    iiko_product_id TEXT REFERENCES iiko_products(id) ON DELETE SET NULL,
    product_name TEXT NOT NULL,
    quantity REAL NOT NULL,
    reserved_quantity REAL NOT NULL DEFAULT 0
);
CREATE INDEX idx_order_items_order_id ON order_items(order_id);
CREATE INDEX idx_order_items_iiko_product_id ON order_items(iiko_product_id);

CREATE TABLE order_history (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    order_id INTEGER NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    changed_by_username TEXT NOT NULL DEFAULT '',
    changed_at DATETIME NOT NULL DEFAULT (strftime('%Y-%m-%d %H:%M:%f+00:00', 'now'))
);
CREATE INDEX idx_order_history_order_id ON order_history(order_id);
CREATE INDEX idx_order_history_changed_at ON order_history(changed_at);

CREATE TABLE order_history_items (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    history_id INTEGER NOT NULL REFERENCES order_history(id) ON DELETE CASCADE,
    change_type TEXT NOT NULL CHECK (change_type IN ('added', 'updated', 'removed')),
    product_code TEXT NOT NULL DEFAULT '',
    product_name TEXT NOT NULL DEFAULT '',
    old_quantity REAL,
    new_quantity REAL,
    old_reserved_quantity REAL,
    new_reserved_quantity REAL
);
CREATE INDEX idx_order_history_items_history_id ON order_history_items(history_id);

CREATE TABLE dish_groups (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    category_id INTEGER REFERENCES order_categories(id) ON DELETE SET NULL,
    name TEXT NOT NULL,
    sort_order INTEGER NOT NULL DEFAULT 0,
    created_at DATETIME NOT NULL DEFAULT (strftime('%Y-%m-%d %H:%M:%f+00:00', 'now')),
    updated_at DATETIME NOT NULL DEFAULT (strftime('%Y-%m-%d %H:%M:%f+00:00', 'now')),
    UNIQUE (category_id, name)
);

CREATE TABLE dish_catalog (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    code TEXT NOT NULL,
    name TEXT NOT NULL,
    created_at DATETIME NOT NULL DEFAULT (strftime('%Y-%m-%d %H:%M:%f+00:00', 'now')),
    updated_at DATETIME NOT NULL DEFAULT (strftime('%Y-%m-%d %H:%M:%f+00:00', 'now')),
    sort_order INTEGER NOT NULL DEFAULT 0,
    category_id INTEGER REFERENCES order_categories(id) ON DELETE SET NULL,
    group_id INTEGER REFERENCES dish_groups(id) ON DELETE SET NULL
);
CREATE UNIQUE INDEX idx_dish_catalog_code ON dish_catalog(code);
CREATE INDEX idx_dish_catalog_sort_order ON dish_catalog(sort_order, id);
CREATE INDEX idx_dish_catalog_group_id ON dish_catalog(group_id);

CREATE TABLE iiko_sync_runs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    source TEXT NOT NULL,
    date_from TEXT NOT NULL DEFAULT '',
    date_to TEXT NOT NULL DEFAULT '',
    known_revision INTEGER NOT NULL DEFAULT -1,
    status TEXT NOT NULL,
    error TEXT NOT NULL DEFAULT '',
    started_at DATETIME NOT NULL DEFAULT (strftime('%Y-%m-%d %H:%M:%f+00:00', 'now')),
    finished_at DATETIME
);

CREATE TABLE iiko_assembly_charts (
    id TEXT PRIMARY KEY,
    assembled_product_id TEXT NOT NULL,
    date_from TEXT NOT NULL,
    date_to TEXT,
    assembled_amount REAL NOT NULL DEFAULT 0,
    product_writeoff_strategy TEXT NOT NULL DEFAULT '',
    product_size_assembly_strategy TEXT NOT NULL DEFAULT '',
    effective_direct_writeoff_store_spec_json TEXT NOT NULL DEFAULT '',
    raw_json TEXT NOT NULL,
    updated_at DATETIME NOT NULL DEFAULT (strftime('%Y-%m-%d %H:%M:%f+00:00', 'now'))
);
CREATE INDEX idx_iiko_assembly_charts_product ON iiko_assembly_charts(assembled_product_id);
CREATE INDEX idx_iiko_assembly_charts_period ON iiko_assembly_charts(date_from, date_to);

CREATE TABLE iiko_assembly_chart_items (
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
CREATE INDEX idx_iiko_assembly_chart_items_chart ON iiko_assembly_chart_items(chart_id);
CREATE INDEX idx_iiko_assembly_chart_items_product ON iiko_assembly_chart_items(product_id);

CREATE TABLE iiko_prepared_charts (
    id TEXT PRIMARY KEY,
    assembled_product_id TEXT NOT NULL,
    date_from TEXT NOT NULL,
    date_to TEXT,
    product_size_assembly_strategy TEXT NOT NULL DEFAULT '',
    effective_direct_writeoff_store_spec_json TEXT NOT NULL DEFAULT '',
    raw_json TEXT NOT NULL,
    updated_at DATETIME NOT NULL DEFAULT (strftime('%Y-%m-%d %H:%M:%f+00:00', 'now'))
);
CREATE INDEX idx_iiko_prepared_charts_product ON iiko_prepared_charts(assembled_product_id);
CREATE INDEX idx_iiko_prepared_charts_period ON iiko_prepared_charts(date_from, date_to);

CREATE TABLE iiko_prepared_chart_items (
    id TEXT PRIMARY KEY,
    prepared_chart_id TEXT NOT NULL REFERENCES iiko_prepared_charts(id) ON DELETE CASCADE,
    sort_weight REAL NOT NULL DEFAULT 0,
    product_id TEXT NOT NULL,
    product_size_spec_json TEXT NOT NULL DEFAULT '',
    store_spec_json TEXT NOT NULL DEFAULT '',
    amount REAL NOT NULL DEFAULT 0,
    raw_json TEXT NOT NULL
);
CREATE INDEX idx_iiko_prepared_chart_items_chart ON iiko_prepared_chart_items(prepared_chart_id);
CREATE INDEX idx_iiko_prepared_chart_items_product ON iiko_prepared_chart_items(product_id);

CREATE TABLE production_sheets (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    created_by_username TEXT NOT NULL DEFAULT '',
    created_at DATETIME NOT NULL DEFAULT (strftime('%Y-%m-%d %H:%M:%f+00:00', 'now')),
    updated_at DATETIME NOT NULL DEFAULT (strftime('%Y-%m-%d %H:%M:%f+00:00', 'now'))
);

CREATE TABLE production_sheet_items (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    sheet_id INTEGER NOT NULL REFERENCES production_sheets(id) ON DELETE CASCADE,
    order_id INTEGER NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    product_name TEXT NOT NULL,
    produced_quantity REAL NOT NULL,
    reason TEXT NOT NULL DEFAULT ''
);
CREATE INDEX idx_production_sheet_items_sheet ON production_sheet_items(sheet_id);
CREATE INDEX idx_production_sheet_items_order ON production_sheet_items(order_id, sheet_id);

CREATE TABLE production_sheet_orders (
    sheet_id INTEGER NOT NULL REFERENCES production_sheets(id) ON DELETE CASCADE,
    order_id INTEGER NOT NULL UNIQUE REFERENCES orders(id) ON DELETE CASCADE,
    PRIMARY KEY (sheet_id, order_id)
);

CREATE TABLE production_sheet_loads (
    sheet_id INTEGER NOT NULL REFERENCES production_sheets(id) ON DELETE CASCADE,
    order_id INTEGER NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    product_name TEXT NOT NULL,
    loaded_quantity REAL NOT NULL CHECK (loaded_quantity >= 0),
    PRIMARY KEY (sheet_id, order_id, product_name)
);
CREATE INDEX production_sheet_loads_order_id_idx ON production_sheet_loads(order_id);

CREATE TABLE order_drafts (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    created_by_username TEXT NOT NULL,
    category_id INTEGER NOT NULL REFERENCES order_categories(id) ON DELETE CASCADE,
    from_department_id INTEGER REFERENCES departments(id) ON DELETE SET NULL,
    fulfillment_date TEXT NOT NULL CHECK (fulfillment_date = date(fulfillment_date)),
    comments TEXT,
    items TEXT NOT NULL DEFAULT '[]',
    updated_at DATETIME NOT NULL DEFAULT (strftime('%Y-%m-%d %H:%M:%f+00:00', 'now')),
    UNIQUE (created_by_username, category_id)
);

CREATE TABLE order_outbox (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    aggregate_id TEXT NOT NULL,
    event_type TEXT NOT NULL,
    payload TEXT NOT NULL,
    correlation_id TEXT NOT NULL DEFAULT '',
    created_at DATETIME NOT NULL DEFAULT (strftime('%Y-%m-%d %H:%M:%f+00:00', 'now')),
    published_at DATETIME
);

CREATE INDEX idx_order_outbox_unpublished ON order_outbox(id) WHERE published_at IS NULL;

-- +goose Down
DROP TABLE order_outbox;
DROP TABLE order_drafts;
DROP TABLE production_sheet_loads;
DROP TABLE production_sheet_orders;
DROP TABLE production_sheet_items;
DROP TABLE production_sheets;
DROP TABLE iiko_prepared_chart_items;
DROP TABLE iiko_prepared_charts;
DROP TABLE iiko_assembly_chart_items;
DROP TABLE iiko_assembly_charts;
DROP TABLE iiko_sync_runs;
DROP TABLE dish_catalog;
DROP TABLE dish_groups;
DROP TABLE order_history_items;
DROP TABLE order_history;
DROP TABLE order_items;
DROP TABLE iiko_products;
DROP TABLE orders;
DROP TABLE order_counters;
DROP TABLE order_categories;
DROP TABLE auth_users;
DROP TABLE departments;
