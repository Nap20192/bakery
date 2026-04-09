-- 0001_init.up.sql
-- Схема для расчёта количества теста по заявкам из Telegram и ТТК (iiko/excel).

BEGIN;

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- =========================================================
-- Справочники
-- =========================================================

-- Клиенты (заказчики, пишущие заявки в Telegram)
CREATE TABLE clients (
    id               uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    name             text NOT NULL,
    telegram_chat_id bigint UNIQUE,
    phone            text,
    note             text,
    created_at       timestamptz NOT NULL DEFAULT now(),
    updated_at       timestamptz NOT NULL DEFAULT now(),
    deleted_at       timestamptz
);

-- Кухни / пекарни (производственные точки)
CREATE TABLE kitchens (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    name       text NOT NULL,
    address    text,
    iiko_id    text UNIQUE,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    deleted_at timestamptz
);

-- Ингредиенты (включая тесто как отдельный ингредиент/полуфабрикат)
CREATE TABLE ingredients (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    iiko_id    text UNIQUE,
    name       text NOT NULL,
    unit       text NOT NULL, -- кг, г, л, мл, шт
    is_dough   boolean NOT NULL DEFAULT false, -- признак, что это "тесто"
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    deleted_at timestamptz
);

-- Готовая продукция (пирожки, хлеб и т.п.)
CREATE TABLE products (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    iiko_id    text UNIQUE,
    name       text NOT NULL,
    unit       text NOT NULL DEFAULT 'шт', -- шт / кг
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    deleted_at timestamptz
);

-- =========================================================
-- ТТК (технико-технологические карты)
-- =========================================================

CREATE TABLE tech_cards (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    product_id  uuid NOT NULL REFERENCES products(id) ON DELETE RESTRICT,
    version     int  NOT NULL DEFAULT 1,
    source      text NOT NULL CHECK (source IN ('iiko','excel','manual')),
    source_ref  text, -- id выгрузки/имя файла excel / id карты в iiko
    yield_qty   numeric(14,4) NOT NULL CHECK (yield_qty > 0), -- выход готового продукта
    yield_unit  text NOT NULL DEFAULT 'шт',
    valid_from  timestamptz NOT NULL DEFAULT now(),
    valid_to    timestamptz,
    created_at  timestamptz NOT NULL DEFAULT now(),
    updated_at  timestamptz NOT NULL DEFAULT now(),
    deleted_at  timestamptz,
    UNIQUE (product_id, version)
);

CREATE INDEX idx_tech_cards_product_active
    ON tech_cards(product_id)
    WHERE deleted_at IS NULL AND valid_to IS NULL;

-- Состав ТТК
CREATE TABLE tech_card_items (
    id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tech_card_id  uuid NOT NULL REFERENCES tech_cards(id) ON DELETE CASCADE,
    ingredient_id uuid NOT NULL REFERENCES ingredients(id) ON DELETE RESTRICT,
    gross_qty     numeric(14,4) NOT NULL CHECK (gross_qty >= 0),
    net_qty       numeric(14,4) NOT NULL CHECK (net_qty >= 0),
    unit          text NOT NULL,
    created_at    timestamptz NOT NULL DEFAULT now(),
    updated_at    timestamptz NOT NULL DEFAULT now(),
    UNIQUE (tech_card_id, ingredient_id)
);

-- =========================================================
-- Заявки из Telegram
-- =========================================================

CREATE TABLE orders (
    id                   uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    client_id            uuid NOT NULL REFERENCES clients(id) ON DELETE RESTRICT,
    kitchen_id           uuid REFERENCES kitchens(id) ON DELETE SET NULL,
    telegram_chat_id     bigint,
    telegram_message_id  bigint,
    order_date           date NOT NULL,            -- дата на которую нужна продукция
    status               text NOT NULL DEFAULT 'new'
                                CHECK (status IN ('new','parsed','confirmed','processed','cancelled')),
    raw_text             text,                     -- исходное сообщение
    note                 text,
    created_at           timestamptz NOT NULL DEFAULT now(),
    updated_at           timestamptz NOT NULL DEFAULT now(),
    deleted_at           timestamptz,
    UNIQUE (telegram_chat_id, telegram_message_id)
);

CREATE INDEX idx_orders_date ON orders(order_date);
CREATE INDEX idx_orders_client ON orders(client_id);

CREATE TABLE order_items (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id    uuid NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    product_id  uuid NOT NULL REFERENCES products(id) ON DELETE RESTRICT,
    quantity    numeric(14,4) NOT NULL CHECK (quantity > 0),
    unit        text NOT NULL DEFAULT 'шт',
    created_at  timestamptz NOT NULL DEFAULT now(),
    updated_at  timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX idx_order_items_order ON order_items(order_id);
CREATE INDEX idx_order_items_product ON order_items(product_id);

-- =========================================================
-- Расчёт теста
-- =========================================================

CREATE TABLE dough_calculations (
    id                   uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    kitchen_id           uuid REFERENCES kitchens(id) ON DELETE SET NULL,
    calc_date            date NOT NULL,   -- на какую дату считаем
    dough_ingredient_id  uuid NOT NULL REFERENCES ingredients(id) ON DELETE RESTRICT,
    total_qty            numeric(14,4) NOT NULL CHECK (total_qty >= 0),
    unit                 text NOT NULL,
    source               text NOT NULL DEFAULT 'auto', -- auto/manual
    created_at           timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX idx_dough_calc_date ON dough_calculations(calc_date);

-- Детализация: сколько теста ушло на каждую позицию заявки
CREATE TABLE dough_calculation_items (
    id              uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    calculation_id  uuid NOT NULL REFERENCES dough_calculations(id) ON DELETE CASCADE,
    order_item_id   uuid NOT NULL REFERENCES order_items(id) ON DELETE RESTRICT,
    product_id      uuid NOT NULL REFERENCES products(id) ON DELETE RESTRICT,
    tech_card_id    uuid NOT NULL REFERENCES tech_cards(id) ON DELETE RESTRICT,
    quantity        numeric(14,4) NOT NULL, -- количество продукции из заявки
    dough_qty       numeric(14,4) NOT NULL, -- сколько теста нужно
    unit            text NOT NULL,
    created_at      timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX idx_dough_calc_items_calc ON dough_calculation_items(calculation_id);

-- =========================================================
-- Автообновление updated_at
-- =========================================================

CREATE OR REPLACE FUNCTION set_updated_at()
RETURNS trigger AS $$
BEGIN
    NEW.updated_at := now();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DO $$
DECLARE
    t text;
BEGIN
    FOR t IN
        SELECT unnest(ARRAY[
            'clients','kitchens','ingredients','products',
            'tech_cards','tech_card_items','orders','order_items'
        ])
    LOOP
        EXECUTE format(
            'CREATE TRIGGER trg_%1$s_updated_at
             BEFORE UPDATE ON %1$s
             FOR EACH ROW EXECUTE FUNCTION set_updated_at()', t);
    END LOOP;
END$$;

-- =========================================================
-- История изменений (audit) для ключевых таблиц
-- =========================================================

CREATE TABLE audit_log (
    id          bigserial PRIMARY KEY,
    table_name  text        NOT NULL,
    row_id      uuid        NOT NULL,
    operation   text        NOT NULL CHECK (operation IN ('INSERT','UPDATE','DELETE')),
    changed_at  timestamptz NOT NULL DEFAULT now(),
    changed_by  text,
    old_data    jsonb,
    new_data    jsonb
);

CREATE INDEX idx_audit_log_table_row ON audit_log(table_name, row_id);
CREATE INDEX idx_audit_log_changed_at ON audit_log(changed_at DESC);

CREATE OR REPLACE FUNCTION write_audit_log()
RETURNS trigger AS $$
DECLARE
    v_row_id uuid;
BEGIN
    IF TG_OP = 'DELETE' THEN
        v_row_id := (row_to_json(OLD)->>'id')::uuid;
        INSERT INTO audit_log(table_name, row_id, operation, changed_by, old_data, new_data)
        VALUES (TG_TABLE_NAME, v_row_id, TG_OP, current_user, to_jsonb(OLD), NULL);
        RETURN OLD;
    ELSIF TG_OP = 'UPDATE' THEN
        v_row_id := (row_to_json(NEW)->>'id')::uuid;
        INSERT INTO audit_log(table_name, row_id, operation, changed_by, old_data, new_data)
        VALUES (TG_TABLE_NAME, v_row_id, TG_OP, current_user, to_jsonb(OLD), to_jsonb(NEW));
        RETURN NEW;
    ELSE
        v_row_id := (row_to_json(NEW)->>'id')::uuid;
        INSERT INTO audit_log(table_name, row_id, operation, changed_by, old_data, new_data)
        VALUES (TG_TABLE_NAME, v_row_id, TG_OP, current_user, NULL, to_jsonb(NEW));
        RETURN NEW;
    END IF;
END;
$$ LANGUAGE plpgsql;

DO $$
DECLARE
    t text;
BEGIN
    FOR t IN
        SELECT unnest(ARRAY[
            'clients','kitchens','ingredients','products',
            'tech_cards','tech_card_items',
            'orders','order_items',
            'dough_calculations','dough_calculation_items'
        ])
    LOOP
        EXECUTE format(
            'CREATE TRIGGER trg_%1$s_audit
             AFTER INSERT OR UPDATE OR DELETE ON %1$s
             FOR EACH ROW EXECUTE FUNCTION write_audit_log()', t);
    END LOOP;
END$$;

COMMIT;
