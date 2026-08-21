# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

```bash
# Backend
make build      # go build ./...
make vet        # go vet ./...
make test       # go test ./...
make lint       # golangci-lint run ./...
make sqlc       # sqlc generate  — run after editing queries/

# Run a single test
go test ./internal/services/order/... -run TestFoo

# Frontend
go test ./frontend/...
go vet ./frontend/...
FRONTEND_ADDR=:5173 BACKEND_URL=http://127.0.0.1:8080 go run ./frontend
```

After editing any `.sql` file in `queries/`, run `make sqlc` before building.

## Architecture

**Modular monolith** with clean architecture and dependency inversion. Three binaries share one Go module:

- `cmd/worker` — HTTP API + iiko sync + order cleanup; applies DB migrations on startup, creates admin user
- `cmd/bot` — Telegram bot + RabbitMQ order-event consumer
- `frontend` — Go HTML/HTMX BFF; calls the worker API and never the database

**Dependency direction is strictly inward:**

```
delivery (infra/http, inbound/bot)  ─┐
infra/repo (adapters)               ─┤──►  usecase (ports + service)  ──►  domain
composition (app/, deps)            ─┘
```

## Service layout

Every service lives under `internal/services/<svc>/` with this structure:

```
domain/          pure business logic, entities, domain events (no infra imports)
usecase/<svc>/   interfaces.go (UseCase + Repository ports + DTOs) + service impl
infra/repo/      Repository adapter over sqlc
infra/http/      HTTP handler + RegisterRoutes (only if it has HTTP endpoints)
app/app.go       New(...) wires usecase + repo
```

Not every service has every folder: `admin` has no domain/repo; `sync` has no domain/http; `techcard` has no http.

**Package naming convention:**
- `usecase/<svc>` → `package <svc>uc` (import alias `<svc>uc`)
- `domain/` → `package <svc>` (import alias `<svc>domain`)
- `infra/repo/` → `package <svc>repo`
- `infra/http/` → `package <svc>http`
- `app/` → `package <svc>app`

## Composition root

Two-phase wiring in `internal/deps`:

1. `InfraDeps` — db, sqlc `*Queries`, RabbitMQ publisher/consumer, iiko client
2. `AppDeps` — services + transport, each `WithXService(infra)` delegates to `<svc>app.New(...)`

The two binaries select different option sets. Dependency order matters: declare a service after its dependencies (e.g. `admin` needs `auth` + `department`).

## Critical rules

**Never leak sqlc past `infra/repo`.** Repos map sqlc rows to domain/DTO types. Usecase signatures use domain types only.

**Ports live with the consumer.** `usecase/<svc>/interfaces.go` declares `UseCase`, `Repository`, `EventPublisher`, etc. that the service needs. Delivery talks to `UseCase`; the service talks to `Repository`.

**Domain is pure.** No DB, pgx, sqlc, HTTP, telebot, or other service imports in `domain/`.

**Use `internal/pkg/enum`** for roles, department types, iiko product types, rabbitmq names. Never hardcode their string values.

**Never expose raw DB/iiko errors to users.** Log the technical cause; return a safe message.

## Documentation

**Docs are the source of truth for behavior — read them before coding, update them after.** The specs describe how the system must behave; the code must match them, not the other way around.

- `docs/api/openapi.yaml` — HTTP contract. Route-sync test `internal/inbound/api/openapi_test.go` fails if a route is missing here.
- `frontend/FRONTEND_BEHAVIOR.md` — full route + behavior spec for the BFF.
- `CLAUDE.md` (this file) — architecture, conventions, domain rules.

Every behavior change updates the relevant doc **in the same change** as the code. If a doc and the code disagree, treat it as a bug: fix whichever is wrong and reconcile them — never leave them drifted. Use the `sdd` skill for spec-first work (write/adjust the spec, then the code) and to audit a spec against the implementation.

## Authentication

Every request authenticates via `Authorization` header:

- `tma <initData>` — Telegram Mini App: HMAC-validated initData, resolved strictly by `telegram_id` (username never authenticates; unbound accounts bind via the `/login` password fallback)
- `Bearer <token>` — Web: HMAC session token from `POST /login`, resolved by `user_id`

Three middlewares in `internal/inbound/api/httpx/`:
- `RequireMiniAppAuth` — authenticated + attached to shop/workshop department
- `RequireAuth` — any logged-in user
- `RequireAdmin` — admin role only

## Events (RabbitMQ)

Order service records domain events (`order.created`, `order.updated`) on the aggregate via `sharedkernel.AggregateRoot`, then the usecase publishes them through its `EventPublisher` port. Fanout exchange `bakery.order-events` → bot queue `bakery.bot.order-events`. Bot notifies: order creator, all bakers, workshop group chat.

## Database

- `queries/*.sql` → run `sqlc generate` → `internal/outbound/db/sqlc/`
- Migrations in `migrations/NNNNN_name.sql` (goose-style); applied automatically by worker on startup
- `DATABASE_URL` takes priority over individual `POSTGRES_*` vars
- Seed data for dish catalog: `templates/dishes.txt` → «Булочки», `templates/bread.txt` → «Хлеб» (worker сидит оба на старте; категория, назначенная админом, и ручной `sort_order` не затираются — `sort_order` задаётся шаблоном только при первой вставке). Формат шаблона: строка-заголовок «группа» (заглавными) + строки `<код> <название>`.
- Normalized catalog model: `order_categories` → `dish_groups` (группа, FK на категорию) → `dish_catalog` (блюдо, FK `group_id` + `category_id`). Слой repo мапит `group_id` обратно в строку «группа» (поле `Theme` в domain/DTO осталось — это имя группы).

## Domain rules (do not regress)

**Departments:** `shop` and `workshop` types. Orders flow shop → workshop. Shop users see/edit only their own orders; workshop (`baker`) sees all and can also create orders — the source is always resolved server-side to the real «Цех Пекари» department (`GetByCode(ctx, "pekari")`), any client-supplied `from_department_id` is ignored for that role. Own letter/counter sequence (`Ц`), no collision with shop counters.

**Order categories (типы заявок):** dynamic `order_categories` dictionary (letter + color slug from `orderdomain.CategoryColors`). Every order requires a category; its letter is part of the order number (`Г.Х.08.07.26.001`) and must be unique across categories (`orders.number` is UNIQUE — a shared letter would mint colliding numbers), counters are per shop+category+day, category never changes on edit. Dish catalog entries carry `category_id` + `group_id` (нормализованная «группа», см. `dish_groups`).

**Order drafts (черновики):** shop-only convenience on `/orders/new`, keyed by `(created_by_username, category_id)` — one draft per user per category, `order_drafts` table upserts on that pair (`SaveOrderDraft`), never touching order numbering/counters. Same validation as `CreateOrder` (`Service.validateOrderWrite` — shared, extracted). Consumed (deleted) when the real order is created from it; otherwise purged by the same cleanup ticker that prunes old orders (`DeleteOrderDraftsOlderThan`, same retention).

**RBAC roles:** `admin`, `shop`, `baker` (+ `sync` capability). No self-registration; admin panel is the only way to create users. Passwords are pbkdf2-hashed — admin resets, never reveals.

**Отработка (production fact):** лист фиксирует **партию** — сохранённый выбор заказов (`production_sheet_orders`, включая заказы без отклонений) плюс отклонения факта (`production_sheet_items`). Журнал — **единственное место хранения факта; заказ при отработке не изменяется вообще** (read-time декоратор: `decorateProductionFacts` накладывает факт на позиции при чтении, свежий лист побеждает). Связь: **у отработки много заказов, у заказа — максимум одна отработка** — принадлежность определяется выбором партии (конфликт `order.production_exists`). Лист без отклонений валиден и живёт до явного удаления («все совпали» больше не удаляет документ). В историю заказа отработка не пишет; события `order.produced`/`order.production_cleared` — только при реальном изменении видимого факта. CRUD `/production` — только baker/admin. UI: редактор листа один и тот же при создании и правке (журнал), внутри листа — «Расчёт теста» по партии (мониторинг считает по `EffectiveQuantity()` — факт учитывается автоматически).

**Monitor (ingredient calculation):** Business math lives in `monitor/domain`; the usecase loads the tech-card graph and calls it. Two formulas:
- Assembly chart: `scale = ordered_qty / assembled_amount`, `child = item.amount_in × scale`
- Prepared chart: `child = item.amount × ordered_qty`

Keep cycle protection and recursion-depth limits.

## Frontend

Go `html/template` + vendored HTMX + plain CSS/JavaScript in `frontend/`.
Deployed on Railway as a separate Go BFF that calls the worker API over the
private network. API credentials remain in `HttpOnly` cookies.

Role-based UI: `shop` creates/edits orders; `baker`/`workshop` also creates orders (self-sourced from «Цех Пекари», no draft support) plus views all orders and runs ingredient calculations; `admin` manages users. See `frontend/FRONTEND_BEHAVIOR.md` for full route and behavior spec.

Frontend validation (past-date guard) is a UX hint only — backend enforces all rules.

## How to extend

**Add an endpoint:** Add method to `UseCase` interface → implement in service + repo if needed → add handler method + route in `infra/http/handler.go` → describe it in `docs/api/openapi.yaml` (the route-sync test `internal/inbound/api/openapi_test.go` fails otherwise) → update `internal/inbound/api/contract`, the frontend backend adapter and its Query/Command port → run `make frontend-check`.

**Add a service:** domain → usecase (interfaces + impl) → infra/repo → infra/http → app/app.go → wire in `internal/deps` → add handler to `api.NewServer` if it has HTTP routes.

**Add a domain event:** Define in `<svc>/domain` embedding `sharedkernel.Event`; record on aggregate; publish from usecase via `EventPublisher` port; add exchange/queue names to `internal/pkg/enum/rabbitmq.go`; handle in `internal/inbound/bot`.
