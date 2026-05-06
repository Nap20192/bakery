DB_PATH ?= bakery.db
GOOSE ?= goose
GOOSE_DRIVER ?= sqlite3
MIGRATIONS_DIR ?= migrations

.PHONY: migrate-up migrate-down migrate-status migrate-reset

migrate-up:
	$(GOOSE) -dir $(MIGRATIONS_DIR) $(GOOSE_DRIVER) "$(DB_PATH)" up

migrate-down:
	$(GOOSE) -dir $(MIGRATIONS_DIR) $(GOOSE_DRIVER) "$(DB_PATH)" down

migrate-status:
	$(GOOSE) -dir $(MIGRATIONS_DIR) $(GOOSE_DRIVER) "$(DB_PATH)" status

migrate-reset:
	$(GOOSE) -dir $(MIGRATIONS_DIR) $(GOOSE_DRIVER) "$(DB_PATH)" reset
