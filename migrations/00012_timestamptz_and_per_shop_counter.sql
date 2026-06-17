-- +goose Up
-- 1. Convert TEXT timestamp columns to real timestamptz.
--    The `col::timestamptz` cast accepts both formats that exist in the data:
--    the Postgres `now()::text` default (space separator) and the RFC3339
--    values written by the application (T separator).

ALTER TABLE auth_users ALTER COLUMN created_at DROP DEFAULT;
ALTER TABLE auth_users ALTER COLUMN created_at TYPE timestamptz USING created_at::timestamptz;
ALTER TABLE auth_users ALTER COLUMN created_at SET DEFAULT now();
ALTER TABLE auth_users ALTER COLUMN updated_at DROP DEFAULT;
ALTER TABLE auth_users ALTER COLUMN updated_at TYPE timestamptz USING updated_at::timestamptz;
ALTER TABLE auth_users ALTER COLUMN updated_at SET DEFAULT now();

ALTER TABLE departments ALTER COLUMN created_at DROP DEFAULT;
ALTER TABLE departments ALTER COLUMN created_at TYPE timestamptz USING created_at::timestamptz;
ALTER TABLE departments ALTER COLUMN created_at SET DEFAULT now();
ALTER TABLE departments ALTER COLUMN updated_at DROP DEFAULT;
ALTER TABLE departments ALTER COLUMN updated_at TYPE timestamptz USING updated_at::timestamptz;
ALTER TABLE departments ALTER COLUMN updated_at SET DEFAULT now();

ALTER TABLE orders ALTER COLUMN created_at TYPE timestamptz USING created_at::timestamptz;
ALTER TABLE orders ALTER COLUMN created_at SET DEFAULT now();

ALTER TABLE order_history ALTER COLUMN changed_at DROP DEFAULT;
ALTER TABLE order_history ALTER COLUMN changed_at TYPE timestamptz USING changed_at::timestamptz;
ALTER TABLE order_history ALTER COLUMN changed_at SET DEFAULT now();

ALTER TABLE iiko_products ALTER COLUMN updated_at DROP DEFAULT;
ALTER TABLE iiko_products ALTER COLUMN updated_at TYPE timestamptz USING updated_at::timestamptz;
ALTER TABLE iiko_products ALTER COLUMN updated_at SET DEFAULT now();

ALTER TABLE iiko_assembly_charts ALTER COLUMN updated_at DROP DEFAULT;
ALTER TABLE iiko_assembly_charts ALTER COLUMN updated_at TYPE timestamptz USING updated_at::timestamptz;
ALTER TABLE iiko_assembly_charts ALTER COLUMN updated_at SET DEFAULT now();

ALTER TABLE iiko_prepared_charts ALTER COLUMN updated_at DROP DEFAULT;
ALTER TABLE iiko_prepared_charts ALTER COLUMN updated_at TYPE timestamptz USING updated_at::timestamptz;
ALTER TABLE iiko_prepared_charts ALTER COLUMN updated_at SET DEFAULT now();

ALTER TABLE iiko_sync_runs ALTER COLUMN started_at DROP DEFAULT;
ALTER TABLE iiko_sync_runs ALTER COLUMN started_at TYPE timestamptz USING started_at::timestamptz;
ALTER TABLE iiko_sync_runs ALTER COLUMN started_at SET DEFAULT now();
ALTER TABLE iiko_sync_runs ALTER COLUMN finished_at TYPE timestamptz USING finished_at::timestamptz;

-- 2. orders.fulfillment_date is a calendar date, not a timestamp.
ALTER TABLE orders ALTER COLUMN fulfillment_date DROP DEFAULT;
ALTER TABLE orders ALTER COLUMN fulfillment_date TYPE date USING fulfillment_date::date;
ALTER TABLE orders ALTER COLUMN fulfillment_date SET DEFAULT CURRENT_DATE;

-- 3. dish_catalog timestamps used DEFAULT '' (an unparseable empty string).
--    Drop NOT NULL while rewriting, backfill empties, then restore.
ALTER TABLE dish_catalog ALTER COLUMN created_at DROP DEFAULT;
ALTER TABLE dish_catalog ALTER COLUMN created_at DROP NOT NULL;
ALTER TABLE dish_catalog ALTER COLUMN created_at TYPE timestamptz USING NULLIF(created_at, '')::timestamptz;
UPDATE dish_catalog SET created_at = now() WHERE created_at IS NULL;
ALTER TABLE dish_catalog ALTER COLUMN created_at SET NOT NULL;
ALTER TABLE dish_catalog ALTER COLUMN created_at SET DEFAULT now();

ALTER TABLE dish_catalog ALTER COLUMN updated_at DROP DEFAULT;
ALTER TABLE dish_catalog ALTER COLUMN updated_at DROP NOT NULL;
ALTER TABLE dish_catalog ALTER COLUMN updated_at TYPE timestamptz USING NULLIF(updated_at, '')::timestamptz;
UPDATE dish_catalog SET updated_at = now() WHERE updated_at IS NULL;
ALTER TABLE dish_catalog ALTER COLUMN updated_at SET NOT NULL;
ALTER TABLE dish_catalog ALTER COLUMN updated_at SET DEFAULT now();

-- 4. Make order numbering sequential per shop: the day counter is now keyed
--    by (day, department_id) instead of by day alone.
ALTER TABLE order_counters ADD COLUMN IF NOT EXISTS department_id BIGINT NOT NULL DEFAULT 0;
ALTER TABLE order_counters DROP CONSTRAINT IF EXISTS order_counters_pkey;
ALTER TABLE order_counters ADD PRIMARY KEY (day, department_id);

-- 5. Preserve referential integrity for orders: a department referenced by an
--    order can no longer be deleted out from under it (was ON DELETE SET NULL,
--    which silently orphaned orders and broke per-shop visibility).
ALTER TABLE orders DROP CONSTRAINT IF EXISTS orders_from_department_id_fkey;
ALTER TABLE orders ADD CONSTRAINT orders_from_department_id_fkey
    FOREIGN KEY (from_department_id) REFERENCES departments(id) ON DELETE RESTRICT;
ALTER TABLE orders DROP CONSTRAINT IF EXISTS orders_to_department_id_fkey;
ALTER TABLE orders ADD CONSTRAINT orders_to_department_id_fkey
    FOREIGN KEY (to_department_id) REFERENCES departments(id) ON DELETE RESTRICT;

-- +goose Down
ALTER TABLE orders DROP CONSTRAINT IF EXISTS orders_to_department_id_fkey;
ALTER TABLE orders ADD CONSTRAINT orders_to_department_id_fkey
    FOREIGN KEY (to_department_id) REFERENCES departments(id) ON DELETE SET NULL;
ALTER TABLE orders DROP CONSTRAINT IF EXISTS orders_from_department_id_fkey;
ALTER TABLE orders ADD CONSTRAINT orders_from_department_id_fkey
    FOREIGN KEY (from_department_id) REFERENCES departments(id) ON DELETE SET NULL;

ALTER TABLE order_counters DROP CONSTRAINT IF EXISTS order_counters_pkey;
ALTER TABLE order_counters DROP COLUMN IF EXISTS department_id;
ALTER TABLE order_counters ADD PRIMARY KEY (day);

ALTER TABLE dish_catalog ALTER COLUMN updated_at DROP DEFAULT;
ALTER TABLE dish_catalog ALTER COLUMN updated_at TYPE text USING updated_at::text;
ALTER TABLE dish_catalog ALTER COLUMN updated_at SET DEFAULT '';
ALTER TABLE dish_catalog ALTER COLUMN created_at DROP DEFAULT;
ALTER TABLE dish_catalog ALTER COLUMN created_at TYPE text USING created_at::text;
ALTER TABLE dish_catalog ALTER COLUMN created_at SET DEFAULT '';

ALTER TABLE orders ALTER COLUMN fulfillment_date DROP DEFAULT;
ALTER TABLE orders ALTER COLUMN fulfillment_date TYPE text USING fulfillment_date::text;
ALTER TABLE orders ALTER COLUMN fulfillment_date SET DEFAULT (CURRENT_DATE::text);

ALTER TABLE iiko_sync_runs ALTER COLUMN finished_at TYPE text USING finished_at::text;
ALTER TABLE iiko_sync_runs ALTER COLUMN started_at DROP DEFAULT;
ALTER TABLE iiko_sync_runs ALTER COLUMN started_at TYPE text USING started_at::text;
ALTER TABLE iiko_sync_runs ALTER COLUMN started_at SET DEFAULT (now()::text);

ALTER TABLE iiko_prepared_charts ALTER COLUMN updated_at DROP DEFAULT;
ALTER TABLE iiko_prepared_charts ALTER COLUMN updated_at TYPE text USING updated_at::text;
ALTER TABLE iiko_prepared_charts ALTER COLUMN updated_at SET DEFAULT (now()::text);

ALTER TABLE iiko_assembly_charts ALTER COLUMN updated_at DROP DEFAULT;
ALTER TABLE iiko_assembly_charts ALTER COLUMN updated_at TYPE text USING updated_at::text;
ALTER TABLE iiko_assembly_charts ALTER COLUMN updated_at SET DEFAULT (now()::text);

ALTER TABLE iiko_products ALTER COLUMN updated_at DROP DEFAULT;
ALTER TABLE iiko_products ALTER COLUMN updated_at TYPE text USING updated_at::text;
ALTER TABLE iiko_products ALTER COLUMN updated_at SET DEFAULT (now()::text);

ALTER TABLE order_history ALTER COLUMN changed_at DROP DEFAULT;
ALTER TABLE order_history ALTER COLUMN changed_at TYPE text USING changed_at::text;
ALTER TABLE order_history ALTER COLUMN changed_at SET DEFAULT (now()::text);

ALTER TABLE orders ALTER COLUMN created_at DROP DEFAULT;
ALTER TABLE orders ALTER COLUMN created_at TYPE text USING created_at::text;

ALTER TABLE departments ALTER COLUMN updated_at DROP DEFAULT;
ALTER TABLE departments ALTER COLUMN updated_at TYPE text USING updated_at::text;
ALTER TABLE departments ALTER COLUMN updated_at SET DEFAULT (now()::text);
ALTER TABLE departments ALTER COLUMN created_at DROP DEFAULT;
ALTER TABLE departments ALTER COLUMN created_at TYPE text USING created_at::text;
ALTER TABLE departments ALTER COLUMN created_at SET DEFAULT (now()::text);

ALTER TABLE auth_users ALTER COLUMN updated_at DROP DEFAULT;
ALTER TABLE auth_users ALTER COLUMN updated_at TYPE text USING updated_at::text;
ALTER TABLE auth_users ALTER COLUMN updated_at SET DEFAULT (now()::text);
ALTER TABLE auth_users ALTER COLUMN created_at DROP DEFAULT;
ALTER TABLE auth_users ALTER COLUMN created_at TYPE text USING created_at::text;
ALTER TABLE auth_users ALTER COLUMN created_at SET DEFAULT (now()::text);
