DATABASE_URL ?= postgres://postgres:postgres@localhost:5432/bakery?sslmode=disable
GOOSE ?= goose
GOOSE_DRIVER ?= postgres
MIGRATIONS_DIR ?= migrations

.PHONY: migrate-up migrate-down migrate-status migrate-reset

migrate-up:
	$(GOOSE) -dir $(MIGRATIONS_DIR) $(GOOSE_DRIVER) "$(DATABASE_URL)" up

migrate-down:
	$(GOOSE) -dir $(MIGRATIONS_DIR) $(GOOSE_DRIVER) "$(DATABASE_URL)" down

migrate-status:
	$(GOOSE) -dir $(MIGRATIONS_DIR) $(GOOSE_DRIVER) "$(DATABASE_URL)" status

migrate-reset:
	$(GOOSE) -dir $(MIGRATIONS_DIR) $(GOOSE_DRIVER) "$(DATABASE_URL)" reset
