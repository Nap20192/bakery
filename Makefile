
GOCMD = go

.PHONY: build test lint vet sqlc frontend-check test-orders

build:
	$(GOCMD) build ./...

vet:
	$(GOCMD) vet ./...

test:
	$(GOCMD) test ./...

lint:
	golangci-lint run ./...

sqlc:
	sqlc generate

frontend-check:
	$(GOCMD) test ./frontend/...

# Удаляет ВСЕ заказы и создаёт свежие тестовые (dev-инструмент).
test-orders:
	$(GOCMD) run ./cmd/testingorders -yes
