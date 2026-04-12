CREATE TABLE users (
    id uuid PRIMARY KEY,
    login text NOT NULL UNIQUE,
    password_hash text NOT NULL,
    role text NOT NULL CHECK (role IN ('admin', 'client', 'kitchen')),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE clients (
    id uuid PRIMARY KEY,
    user_id uuid UNIQUE REFERENCES users (id) ON DELETE SET NULL,
    name text NOT NULL,
    telegram_chat_id bigint UNIQUE,
    note text,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE kitchens (
    id uuid PRIMARY KEY,
    user_id uuid UNIQUE REFERENCES users (id) ON DELETE SET NULL,
    name text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE ingredients (
    id uuid PRIMARY KEY,
    iiko_id text UNIQUE,
    name text NOT NULL UNIQUE,
    unit text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE products (
    id uuid PRIMARY KEY,
    iiko_id text UNIQUE,
    name text NOT NULL UNIQUE,
    unit text NOT NULL DEFAULT 'шт',
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE tech_cards (
    id uuid PRIMARY KEY,
    product_id uuid NOT NULL REFERENCES products (id) ON DELETE RESTRICT,
    yield_qty numeric(14, 4) NOT NULL CHECK (yield_qty > 0),
    yield_unit text NOT NULL DEFAULT 'шт',
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE tech_card_items (
    id uuid PRIMARY KEY,
    tech_card_id uuid NOT NULL REFERENCES tech_cards (id) ON DELETE CASCADE,
    ingredient_id uuid REFERENCES ingredients (id) ON DELETE RESTRICT,
    sub_product_id uuid REFERENCES products (id) ON DELETE RESTRICT,
    gross_qty numeric(14, 4) NOT NULL CHECK (gross_qty > 0),
    net_qty numeric(14, 4) NOT NULL CHECK (net_qty > 0),
    unit text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT item_has_one_source CHECK (
        (
            ingredient_id IS NOT NULL
            AND sub_product_id IS NULL
        )
        OR (
            ingredient_id IS NULL
            AND sub_product_id IS NOT NULL
        )
    )
);

CREATE TABLE orders (
    id uuid PRIMARY KEY,
    client_id uuid NOT NULL REFERENCES clients (id) ON DELETE RESTRICT,
    kitchen_id uuid REFERENCES kitchens (id) ON DELETE SET NULL,
    telegram_chat_id bigint,
    telegram_message_id bigint,
    order_date date NOT NULL,
    status text NOT NULL DEFAULT 'new' CHECK (
        status IN (
            'new',
            'parsed',
            'confirmed',
            'processed',
            'cancelled'
        )
    ),
    raw_text text,
    note text,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE order_items (
    id uuid PRIMARY KEY,
    order_id uuid NOT NULL REFERENCES orders (id) ON DELETE CASCADE,
    product_id uuid NOT NULL REFERENCES products (id) ON DELETE RESTRICT,
    quantity numeric(14, 4) NOT NULL CHECK (quantity > 0),
    unit text NOT NULL DEFAULT 'шт',
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

-- индексы на внешние ключи
CREATE INDEX ON clients (user_id);

CREATE INDEX ON kitchens (user_id);

CREATE INDEX ON tech_cards (product_id);

CREATE INDEX ON tech_card_items (tech_card_id);

CREATE INDEX ON tech_card_items (ingredient_id);

CREATE INDEX ON tech_card_items (sub_product_id);

CREATE INDEX ON orders (client_id);

CREATE INDEX ON orders (kitchen_id);

CREATE INDEX ON order_items (order_id);

CREATE INDEX ON order_items (product_id);
