
GOCMD = go

.PHONY: build test lint vet sqlc api-gen test-orders

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

# Перегенерировать типы фронта из docs/api/openapi.yaml и проверить их.
api-gen:
	cd frontend && npm run api-gen && npm run typecheck

# Удаляет ВСЕ заказы и создаёт свежие тестовые (dev-инструмент).
test-orders:
	$(GOCMD) run ./cmd/testingorders -yes
