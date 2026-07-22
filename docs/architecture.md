# Architecture

The system is a **modular monolith** with clean architecture and strict
dependency inversion. One Go module produces the worker, bot, and frontend
binaries. The worker and bot share service code; the frontend is an HTTP
client of the worker API.

## Technology stack

Backend (`go.mod`, `docker-compose.yml`):

| Component | Version |
|---|---|
| Go | 1.26.1 |
| PostgreSQL | 17 (`postgres:17`) |
| RabbitMQ | 3.x (`rabbitmq:3-management-alpine`) |
| `github.com/jackc/pgx/v5` | v5.9.2 (+ pgxpool) |
| `github.com/rabbitmq/amqp091-go` | v1.12.0 |
| `gopkg.in/telebot.v3` | v3.3.8 |
| `github.com/joho/godotenv` | v1.5.1 |
| `github.com/google/uuid` | v1.6.0 |
| `golang.org/x/sync` (errgroup) | v0.20.0 |
| sqlc | config schema v2 (`sqlc.yaml`); the CLI is installed separately (`sqlc generate`, or `go run github.com/sqlc-dev/sqlc/cmd/sqlc@latest generate`) |
| Migrations | goose-format SQL files, applied by the built-in runner `internal/pkg/dbmigrate` (no external goose binary) |
| Lint | golangci-lint (`.golangci.yml`) |

Frontend (`frontend/`):

| Component | Version |
|---|---|
| Runtime and rendering | Go `net/http`, `html/template`, `embed` |
| Hypermedia | HTMX 2.0.4, vendored in `static/vendor` |
| Styling | plain mobile-first CSS with semantic tokens |
| Browser code | plain JavaScript for Telegram bridge and local interaction state |
| Application boundary | CQRS `Queries` and `Commands` in `frontend/internal/application` |
| API adapter | typed Go package `frontend/internal/backend` |

There is no Node runtime, bundler, package manager, or client framework in the
frontend build.

## Binaries

| Binary | Responsibilities |
|---|---|
| `cmd/worker` | HTTP API, iiko sync loop, order cleanup ticker, outbox relay. Applies DB migrations on startup, ensures the admin user, seeds the dish catalog from `templates/`. |
| `cmd/bot` | Telegram bot (login/info commands, Mini App entry) and the RabbitMQ consumer that turns order events into Telegram notifications. |
| `frontend` | HTML/HTMX BFF. Renders the Mini App, owns browser cookies and CSRF, and calls the worker API over HTTP. |

The bot binary contains **no** HTTP API and the worker contains **no**
Telegram polling; the only channel between them is RabbitMQ (plus the shared
database). The frontend contains no repositories or service use cases and
never accesses the database directly.

## Layering

Dependency direction is strictly inward:

```
delivery (infra/http, inbound/bot)  ─┐
infra/repo (adapters over sqlc)     ─┤──►  usecase (ports + service)  ──►  domain
composition (internal/deps, app/)   ─┘
```

- **domain** — pure business logic: entities, invariants, domain events. No
  imports of DB, pgx, sqlc, HTTP, telebot, or other services.
- **usecase** (`usecase/<svc>/interfaces.go`) — declares the `UseCase`
  boundary that delivery talks to, and the ports the service itself needs
  (`Repository`, `EventPublisher`, …). Ports live with the **consumer**;
  adapters implement them.
- **infra/repo** — adapters over the sqlc-generated query layer. sqlc types
  never leak past this layer; repos map rows to domain/DTO types.
- **infra/http** — HTTP handlers + `RegisterRoutes(mux, auth)`; they depend on
  the `UseCase` interface only.
- **app/app.go** — per-service composition (`<svc>app.New(...)` wires usecase
  + repo).

## Service layout

Every service lives under `internal/services/<svc>/`:

```
domain/          entities, pure services, domain events
usecase/<svc>/   interfaces.go (UseCase + ports + DTOs) + service impl
infra/repo/      Repository adapter over sqlc
infra/http/      Handler + RegisterRoutes (only for services with endpoints)
app/app.go       New(...) wiring
```

Not every service has every folder: `admin` has no domain/repo (it composes
auth + department behind ports); `sync` has no domain/http; `techcard` has no
http.

Package naming: `usecase/<svc>` → `package <svc>uc`, `domain/` → `package
<svc>` (imported as `<svc>domain`), `infra/repo` → `<svc>repo`, `infra/http` →
`<svc>http`, `app/` → `<svc>app`.

## Composition root

Two-phase wiring in `internal/deps`:

1. **`InfraDeps`** (`infra.go`) — config, `pgxpool` connection, sqlc
   `*Queries`, RabbitMQ publisher/consumer, iiko client. Built with
   `NewInfraDeps(WithConfig, WithPostgres, WithRepositories, WithRabbitMQ,
   WithIikoClient)`.
2. **`AppDeps`** (`deps.go`) — services and transport. Each
   `WithXService(infra)` option delegates to `<svc>app.New(...)`.

The two binaries select different option sets. **Declaration order matters**:
a service must appear after its dependencies (e.g. `WithAdminService()` needs
`AuthService` and `DepartmentService` already set). Options fail fast with a
clear error when a dependency is missing.

`api.NewServer` receives the use-cases that have HTTP routes and registers
each handler's `RegisterRoutes` on a single `http.ServeMux`, plus
`GET /health`.

## Frontend CQRS boundary

The BFF has three explicit layers:

```text
web handlers -> application Queries / Commands <- backend HTTP adapter
```

- `application` owns the gateway ports, read/write facades, frontend filters,
  and safe transport errors;
- `backend` implements both ports over the worker HTTP API;
- `web` owns cookies, CSRF, routes, view-model projection and HTML rendering;
- `frontend/main.go` is the composition root and is the only place that wires
  the concrete adapter into Queries and Commands.

Web handlers do not import the backend adapter. The application layer does not
know about HTTP, templates, cookies, or worker implementation details.

## API contract (OpenAPI)

The HTTP API is described in **`docs/api/openapi.yaml`** — the single source
of truth for both sides:

- **Backend**: `internal/inbound/api/openapi_test.go` fails when a route
  registered in the mux is missing from the spec or vice versa (it scans the
  `mux.Handle("METHOD /path"` patterns in the delivery adapters).
- **Shared Go DTOs**: `internal/inbound/api/contract` is used by worker HTTP
  handlers and the frontend backend adapter, so request and response structs
  are not duplicated.
- **Frontend**: `frontend/internal/backend` implements application query and
  command ports. `go test ./frontend/...` compiles the boundary and executes
  every HTML view with typed fixtures.

Changing a handler or DTO: update the spec → `go test ./internal/inbound/api`
→ `make frontend-check`.

## Authentication

Every API request authenticates through the `Authorization` header, handled
by `internal/inbound/api/httpx.Authenticator`:

- **`tma <initData>`** — Telegram Mini App. The initData query string is
  HMAC-SHA256 validated against the bot token (Telegram WebApp scheme),
  rejected when `auth_date` is older than **24 hours** or in the future.
  The user is resolved by `telegram_username` from the initData payload.
- **`Bearer <token>`** — web login. `POST /login` verifies the password and
  issues an HMAC session token (`internal/pkg/authtoken`), signed with the
  bot token as the secret; the user is resolved by `user_id` claim.

Three middlewares gate the routes:

| Middleware | Requires |
|---|---|
| `RequireMiniAppAuth` | Authenticated user with a supported role (`admin`, `shop`, `baker`). Department binding is optional and attached to the viewer when present. |
| `RequireAuth` | Any logged-in user (used by `GET /me`). |
| `RequireAdmin` | The `admin` role. |

Finer-grained rules (e.g. "only shop creates orders", "only baker/admin
writes production sheets") are enforced inside the handlers on top of these
middlewares.

The frontend BFF receives Telegram initData or a web password once, validates
it through the worker, and stores the resulting API credential in an
`HttpOnly`, `SameSite=Lax` cookie. Mutating HTML routes require a separate
CSRF token. Browser code never stores or sends the worker bearer token.

## Events and messaging

The order service is the only event producer. Flow:

1. Domain events (`order.created`, `order.updated`, `order.cancelled`,
   `order.restored`, `order.produced`, `order.production_cleared`) are
   recorded on the `Order` aggregate via `sharedkernel.AggregateRoot`.
2. The repository persists them into the **transactional outbox** table
   (`order_outbox`) atomically with the write.
3. The **outbox relay** (`order/infra/outbox`, runs in the worker) polls
   unpublished rows every `OUTBOX_INTERVAL` (default 2s, batch 100) and
   publishes them to the fanout exchange **`bakery.order-events`**.
   Publish-then-mark gives **at-least-once** delivery — consumers must
   tolerate duplicates.
4. The bot consumes the durable queue **`bakery.bot.order-events`** and
   notifies: the order's creator (DM), every user with the `baker` role (DM),
   and the workshop group chat (`TELEGRAM_WORKSHOP_CHAT_ID`). Sends are
   best-effort — a failed send is logged, never requeued.

Exchange/queue names live in `internal/pkg/enum/rabbitmq.go`; never hardcode
the strings.

## Database

- PostgreSQL via `pgx/v5` + `pgxpool`; queries written in `queries/*.sql` and
  compiled by **sqlc** into `internal/outbound/db/sqlc/`. After editing any
  `.sql` file run `make sqlc` (or `sqlc generate`) before building. Every
  query is documented in [database.md](database.md).
- Migrations in `migrations/NNNNN_name.sql` (goose format), applied
  automatically by the worker on startup (`internal/pkg/dbmigrate`).
- `DATABASE_URL` takes priority; otherwise the DSN is assembled from
  `POSTGRES_*` variables.
- Seed data: the worker seeds the dish catalog on startup —
  `templates/dishes.txt` → category «Булочки», `templates/bread.txt` →
  «Хлеб». Seeding is non-destructive: a category already assigned by the
  admin is never overwritten.

## Shared packages (`internal/pkg`)

| Package | Purpose |
|---|---|
| `apperr` | Transport-agnostic error taxonomy: `*Error{Kind, Code, Message}`. Use cases return `apperr` sentinels; the HTTP layer maps `Kind` → status code in one place. `errors.Is` works by identity even through wrapping. |
| `enum` | Roles, department types, iiko product types, RabbitMQ names. Never hardcode these strings. |
| `authtoken` | HMAC-signed web session tokens (issue/parse). |
| `sharedkernel` | `AggregateRoot`, `DomainEvent`, `Event` base for domain events. |
| `helpers` | pgtype conversion helpers (`Timestamptz`, `DateOf`, …). |
| `correlation` | Correlation-ID propagation for logs and event metadata. |
| `dbmigrate` | Goose-style migration runner used by the worker. |

## Configuration

All configuration comes from environment variables (`.env` is loaded in dev
via `config.LoadDotenv()`; see `.env.example`):

| Area | Variables |
|---|---|
| HTTP | `HTTP_HOST`, `HTTP_PORT` (or `PORT`), `HTTP_ALLOWED_ORIGINS`, `HTTP_SHUTDOWN_TIMEOUT` |
| Database | `DATABASE_URL` **or** `POSTGRES_HOST/PORT/USER/PASSWORD/DB/SSLMODE` |
| RabbitMQ | `RABBITMQ_URL` |
| Telegram | `BOT_ENV` (+ `TEST_BOT_TOKEN`/`PROD_BOT_TOKEN` or `BOT_TOKEN`), `MINI_APP_URL`, `TELEGRAM_WORKSHOP_CHAT_ID` |
| iiko | `IIKO_HOST`, `IIKO_PORT`, `IIKO_LOGIN`, `IIKO_PASSWORD` |
| Admin bootstrap | `ADMIN_USERNAME`, `ADMIN_PASSWORD` |
| Background jobs | `SYNC_INTERVAL`, `ORDER_CLEANUP_INTERVAL`, `ORDER_RETENTION`, `OUTBOX_INTERVAL` |
| Logging | `LOG_LEVEL`, `LOG_PRETTY`, `LOG_DIR` |
| Migrations | `MIGRATIONS_DIR` |

The bot token is dual-use: Telegram API access **and** the HMAC secret for
initData validation and web session tokens.

## Frontend deployment

The frontend (`frontend/`) is a Go HTML/HTMX Telegram Mini App deployed on
Railway as a separate public service. Its BFF calls the worker over Railway's
private network (`http://<service>.railway.internal:8080`), while templates,
CSS, JavaScript, and HTMX are embedded into one static binary. See
[frontend/ui-kit.md](frontend/ui-kit.md) for its interface conventions.

## Error-handling policy

Raw DB/iiko/transport errors are **never** shown to users. Use cases return
`apperr` values with safe Russian messages; handlers log the technical cause
(with correlation ID) and respond with the safe message. The HTTP logging
middleware logs method, path, status, duration, and the error text for failed
requests.
