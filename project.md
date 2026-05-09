# Bakery Project

## Overview

Bakery is a Go service for bakery order intake and iiko synchronization.

The runtime is intentionally single-process:

- `cmd/worker` starts the Telegram bot.
- `cmd/worker` starts periodic iiko synchronization.
- `cmd/worker` ensures the admin user exists on startup.

## Architecture

- `internal/inbound/bot` - Telegram bot handlers and middleware.
- `internal/app` - application services.
- `internal/domain` - business rules and domain models.
- `internal/outbound/db` - PostgreSQL connection.
- `internal/outbound/db/sqlc` - generated sqlc repository code.
- `internal/outbound/iiko` - iiko REST client and DTOs.
- `internal/pkg/dbmigrate` - embedded migration bootstrap.
- `pkg/logger` - slog setup and context-aware logging.

## Storage

PostgreSQL is the only supported runtime database.

Main tables:

- `auth_users`
- `orders`
- `order_items`
- `order_counters`
- `iiko_products`
- `iiko_assembly_charts`
- `iiko_assembly_chart_items`
- `iiko_prepared_charts`
- `iiko_prepared_chart_items`
- `iiko_sync_runs`

## Configuration

Configuration is provided through environment variables.

For Railway, prefer setting `DATABASE_URL` from the PostgreSQL service reference:

```env
DATABASE_URL=${{Postgres.DATABASE_URL}}
```

For local development, `DATABASE_URL` can be empty. The app then builds it from:

- `POSTGRES_USER`
- `POSTGRES_PASSWORD`
- `POSTGRES_HOST`
- `POSTGRES_PORT`
- `POSTGRES_DB`
- `POSTGRES_SSLMODE`

## Commands

```bash
go run ./cmd/worker
go test ./...
go vet ./...
make docker-up
make docker-logs
make docker-down
```

## Deployment

Docker and Railway both run the worker binary:

```bash
/app/bakery
```
