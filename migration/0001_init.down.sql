-- 0001_init.down.sql
BEGIN;

DROP TABLE IF EXISTS dough_calculation_items CASCADE;
DROP TABLE IF EXISTS dough_calculations      CASCADE;
DROP TABLE IF EXISTS order_items             CASCADE;
DROP TABLE IF EXISTS orders                  CASCADE;
DROP TABLE IF EXISTS tech_card_items         CASCADE;
DROP TABLE IF EXISTS tech_cards              CASCADE;
DROP TABLE IF EXISTS products                CASCADE;
DROP TABLE IF EXISTS ingredients             CASCADE;
DROP TABLE IF EXISTS kitchens                CASCADE;
DROP TABLE IF EXISTS clients                 CASCADE;
DROP TABLE IF EXISTS audit_log               CASCADE;

DROP FUNCTION IF EXISTS write_audit_log();
DROP FUNCTION IF EXISTS set_updated_at();

COMMIT;
