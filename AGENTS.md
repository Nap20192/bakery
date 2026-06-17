# Bakery Agent Instructions

## Architecture

- `cmd/worker` is the single application entrypoint.
- `internal/app` contains application services. This layer may use DB, sqlc, iiko client, Telegram dependencies, and domain services.
- `internal/domain` contains pure business logic. It must not import DB, sqlc, pgx, HTTP, Telegram, inbound, outbound, or app packages.
- `internal/inbound` contains inbound adapters: Telegram bot and HTTP API.
- `internal/outbound` contains outbound adapters: PostgreSQL/sqlc and iiko client.
- `internal/pkg` contains internal reusable packages.
- `pkg/logger` contains the shared logger.
- `queries` contains sqlc SQL files.
- `migrations` contains goose-style migrations.

## Layer Rules

- Domain must not import app, outbound, inbound, sqlc, pgx, or telebot.
- App may import domain and outbound.
- Inbound calls app services and handles user input/response formatting.
- Outbound must not know Telegram or app business flows.
- Keep formulas and business calculations in domain services.
- Keep DB/iiko data loading in app/outbound.

## Telegram Bot

- Telegram bot code lives in `internal/inbound/bot`.
- Do not put all handlers into one file.
- Use thematic handler files:
  - `handler_start.go`
  - `handler_auth.go`
  - `handler_order.go`
  - `handler_orders.go`
  - `handler_monitor.go`
  - `handler_techcard.go`
  - `response_builder.go`
- Build Telegram HTML/text responses through `responseBuilder` in `response_builder.go`.
- Do not send monitor automatically after order creation.
- At the end of the order-created message, include a copyable command:
  `/monitor ORDER_NUMBER`

## Departments

- Departments have types: `shop` and `workshop`.
- Default departments:
  - `Магазин Гагарина`
  - `Магазин Сарыарка`
  - `Магазин Шолохова`
  - `Цех Пекари`
- Orders are created from shop to workshop:
  - `from_department_id` = shop that created the order
  - `to_department_id` = `Цех Пекари`
- A user who selected workshop must not create shop orders.
- Users choose location through `/start`.

## Auth And RBAC

- Authorization is required for service commands:
  - `/techcard`
  - `/sync`
- The only effective role for now is `admin`.
- Auth and RBAC are separate services.
- Worker creates the admin user on startup if missing.
- Telegram username is saved in users.
- Telegram ID is a numeric Telegram user identifier used to bind a Telegram account to a DB user.

## Monitor

- Application service: `internal/app/monitor.go`.
- Business calculation: `internal/domain/monitoring/service.go`.
- `MonitorService` loads the tech-card graph from DB and calls the domain service.
- The domain monitoring service performs only business math.
- Do not use `raw_json` for prepared chart calculations; read `iiko_prepared_chart_items`.
- Do not add cache if the user explicitly asks for no cache.
- Formulas:
  - Assembly chart:
    - `scale = ordered_quantity / assembled_amount`
    - `child_amount = item.amount_in * scale`
  - Prepared chart:
    - `child_amount = item.amount * ordered_quantity`
- Assembly chart calculation uses `amount_in`.
- Keep protection against cycles and excessive recursion depth.

## iiko

- iiko client lives in `internal/outbound/iiko`.
- Product groups method:
  - `ListProductGroups(params ProductGroupListParams)`
  - endpoint: `/resto/api/v2/entities/products/group/list`
- iiko tests save results into `internal/outbound/iiko/testdata/results`.
- iiko integration tests must skip when required env vars are not set.

## Database

- PostgreSQL is used through pgx.
- sqlc is used for database queries.
- After changing `queries`, run:
  - `sqlc generate`
- Migrations are goose-style and live in `migrations`.
- Worker applies migrations on startup through `internal/pkg/dbmigrate`.
- `DATABASE_URL` has priority. If empty, config builds it from `POSTGRES_*` env vars.

## Environment

- `BOT_ENV=test|prod`.
- `TEST_BOT_TOKEN` is used for test bot.
- `PROD_BOT_TOKEN` is used for prod bot.
- `LOG_PRETTY=false` by default for deploy.
- On Railway, prefer `DATABASE_URL=${{Postgres.DATABASE_URL}}`.

## Docker And Railway

- There is one main service: `worker`.
- `deploy/Dockerfile.worker` builds `./cmd/worker`.
- `deploy/Dockerfile.bot` builds `./cmd/bot`.
- `railway.json` starts `/app/bakery`.
- On Railway, env vars are configured through Variables. `.env` is not needed there.

## Logging

- Use `pkg/logger`.
- Logs must be structured.
- Do not expose internal DB/iiko errors directly to users.
- Technical causes can go to logs; user responses should use safe messages.

## Enum

- Common enums live in `internal/pkg/enum`.
- Use enums for roles, permissions, department types, iiko product types, and sync statuses.

## Tests

- After code changes, run:
  - `go test ./...`
  - `go vet ./...`
- After SQL changes, run:
  - `sqlc generate`
  - `go test ./...`
- Domain business logic should have unit tests without DB dependencies.

## Change Style

- Do not overwrite unrelated uncommitted user changes.
- Do not delete user files such as `.xls` or `.md` unless explicitly requested.
- Do not do broad refactors outside the task.
- Do not return SQLite.
- Do not add client/baker roles unless explicitly requested.
