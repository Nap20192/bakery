# Architecture

The system is a **modular monolith** with clean architecture and strict
dependency inversion. One Go module produces one binary, `cmd/bakery`, which
runs the API, the background jobs, the Telegram bot and the HTML frontend in a
single process.

## Technology stack

Backend (`go.mod`, `docker-compose.yml`):

| Component | Version |
|---|---|
| Go | 1.26.1 |
| SQLite | `modernc.org/sqlite` v1.62.0 (pure Go, no CGO) |
| `github.com/jackc/pgx/v5` | v5.9.2 — only `cmd/pg2sqlite` (one-time import from the old Postgres) |
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

## Binary

`cmd/bakery` is the only deployable. On startup it imports the old Postgres
data if `IMPORT_POSTGRES_URL` is set and the database file does not exist yet,
applies DB migrations,
ensures the admin user and seeds the dish catalog from `templates/`, then runs
in one process:

| Component | Responsibility |
|---|---|
| HTTP server | The only listener (`PORT`). Serves the HTML/HTMX frontend: Mini App pages, browser cookies, CSRF. |
| API | JSON API (`docs/api/openapi.yaml`). Not mounted on any listener: the frontend calls it through an in-process `http.RoundTripper`, so it is unreachable from the network. |
| iiko sync | Periodic catalog/tech-card sync (`SYNC_INTERVAL`). |
| Order cleanup | Prunes old orders and drafts (`ORDER_CLEANUP_INTERVAL`, `ORDER_RETENTION`). |
| Outbox relay | Delivers persisted order events to the Telegram bot. |
| Telegram bot | Login/info commands, Mini App entry, order notifications. |

The frontend contains no repositories or service use cases and never accesses
the database directly: every read and write goes through the API.

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

1. **`InfraDeps`** (`infra.go`) — config, SQLite `*sql.DB`, sqlc
   `*Queries`, iiko client. Built with
   `NewInfraDeps(WithConfig, WithDatabase, WithRepositories, WithIikoClient)`.
2. **`AppDeps`** (`deps.go`) — services and transport. Each
   `WithXService(infra)` option delegates to `<svc>app.New(...)`.

**Declaration order matters**:
a service must appear after its dependencies (e.g. `WithAdminService()` needs
`AuthService` and `DepartmentService` already set). Options fail fast with a
clear error when a dependency is missing.

`api.NewServer` receives the use-cases that have HTTP routes and registers
each handler's `RegisterRoutes` on a single `http.ServeMux`, plus
`GET /health`, and exposes it as an `http.Handler`.

## Frontend CQRS boundary

The BFF has three explicit layers:

```text
web handlers -> application Queries / Commands <- backend HTTP adapter
```

- `application` owns the gateway ports, read/write facades, frontend filters,
  and safe transport errors;
- `backend` implements both ports over the API, through the in-process
  transport;
- `web` owns cookies, CSRF, routes, view-model projection and HTML rendering;
- `frontend.NewHandler` is the only place that wires the concrete adapter into
  Queries and Commands.

Web handlers do not import the backend adapter. The application layer does not
know about HTTP, templates, cookies, or API implementation details.

## API contract (OpenAPI)

The HTTP API is described in **`docs/api/openapi.yaml`** — the single source
of truth for both sides:

- **Backend**: `internal/inbound/api/openapi_test.go` fails when a route
  registered in the mux is missing from the spec or vice versa (it scans the
  `mux.Handle("METHOD /path"` patterns in the delivery adapters).
- **Shared Go DTOs**: `internal/inbound/api/contract` is used by API HTTP
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
  The user is resolved **strictly by `telegram_id`** from the initData
  payload — the username is mutable and user-controlled, so it is never used
  to authenticate Mini App requests. An account without a bound `telegram_id`
  gets 403 and goes through the Mini App password fallback of `POST /login`,
  which binds the id (see [services/auth.md](services/auth.md)).
- **`Bearer <token>`** — web login. `POST /login` verifies the password and
  issues an HMAC session token (`internal/pkg/authtoken`), signed with the
  bot token as the secret; the user is resolved by `user_id` claim.

Three middlewares gate the routes:

| Middleware | Requires |
|---|---|
| `RequireMiniAppAuth` | Authenticated user with a supported role (`admin`, `shop`, `baker`). Department binding is optional and attached to the viewer when present. |
| `RequireAuth` | Any logged-in user (used by `GET /me`). |
| `RequireAdmin` | The `admin` role. |

Finer-grained rules (shop/baker/admin write orders; baker-created orders use
the workshop as their source; only baker/admin write production sheets) are
enforced inside the handlers on top of these middlewares.

The frontend BFF receives Telegram initData or a web password once, validates
it through the API, and stores the resulting API credential in an
`HttpOnly`, `SameSite=Lax` cookie. Mutating HTML routes require a separate
CSRF token. Browser code never stores or sends the API bearer token.

## Events and messaging

The order service is the only event producer. Flow:

1. Domain events (`order.created`, `order.updated`, `order.cancelled`,
   `order.restored`, `order.produced`, `order.production_cleared`) are
   recorded on the `Order` aggregate via `sharedkernel.AggregateRoot`.
2. The repository persists them into the **transactional outbox** table
   (`order_outbox`) atomically with the write.
3. The **outbox relay** (`order/infra/outbox`) polls unpublished rows every
   `OUTBOX_INTERVAL` (default 2s, batch 100), in id order, and hands each one
   to the bot in-process. Handle-then-mark gives **at-least-once** delivery —
   the handler must tolerate duplicates. There is no message broker.
4. The bot notifies: the order's creator (DM), every user with the `baker`
   role (DM), and the workshop group chat (`TELEGRAM_WORKSHOP_CHAT_ID`). Sends
   are best-effort — a failed send is logged, never retried. An event whose
   payload cannot be decoded is logged and dropped.
5. While the bot is not connected to Telegram (it reconnects every 30s), the
   relay stops at the first undelivered row and retries on the next tick:
   events wait in the outbox and are delivered in order once the bot is up.
   The web UI and the API keep working regardless of Telegram.
6. Without a bot token (allowed only outside `BOT_ENV=prod`) the bot is
   disabled and events are marked delivered without notifications.

## Database

- SQLite in one file (`SQLITE_PATH`, default `data/bakery.db`; on Railway a
  volume at `/data`). Queries written in `queries/*.sql` are compiled by
  **sqlc** (engine `sqlite`) into `internal/outbound/db/sqlc/`. After editing
  any `.sql` file run `make sqlc` before building. Every query and the SQL
  conventions are documented in [database.md](database.md).
- Connection (`internal/outbound/db/sqlite.go`): WAL journal,
  `synchronous=NORMAL`, `busy_timeout=5000`, foreign keys on, transactions
  start with `BEGIN IMMEDIATE`, times written and read in UTC, and **one
  open connection**. Every statement is serialized, so order counters
  cannot race. Inside a transaction, use only the transaction's queries:
  a call through the pool would wait for the connection the transaction
  holds and block until the request times out.
- Unique-constraint violations are recognized by table and column list
  (`outbounddb.IsUniqueViolation`), which is how duplicate orders,
  `order.production_exists` and duplicate logins become domain errors.
- Migrations in `migrations/NNNNN_name.sql` (goose format), applied
  automatically on startup (`internal/pkg/dbmigrate`). `00001_init.sql` is
  the SQLite baseline; history before it lived in Postgres.
- Seed data: the dish catalog is seeded on startup —
  `templates/dishes.txt` → category «Булочки», `templates/bread.txt` →
  «Хлеб». Seeding is non-destructive: a category already assigned by the
  admin is never overwritten.

## Shared packages (`internal/pkg`)

| Package | Purpose |
|---|---|
| `apperr` | Transport-agnostic error taxonomy: `*Error{Kind, Code, Message}`. Use cases return `apperr` sentinels; the HTTP layer maps `Kind` → status code in one place. `errors.Is` works by identity even through wrapping. |
| `enum` | Roles, department types, iiko product types. Never hardcode these strings. |
| `authtoken` | HMAC-signed web session tokens (issue/parse). |
| `sharedkernel` | `AggregateRoot`, `DomainEvent`, `Event` base for domain events. |
| `helpers` | Date helpers (`DateString`, `ParseDate`). |
| `correlation` | Correlation-ID propagation for logs and event metadata. |
| `dbmigrate` | Goose-style migration runner applied on startup. |

## Configuration

All configuration comes from environment variables (`.env` is loaded in dev
via `config.LoadDotenv()`; see `.env.example`):

| Area | Variables |
|---|---|
| HTTP | `HTTP_HOST`, `HTTP_PORT` (or `PORT`), `HTTP_SHUTDOWN_TIMEOUT` |
| Database | `SQLITE_PATH`, `IMPORT_POSTGRES_URL` (one-time import on first start, see [deployment.md](deployment.md#moving-from-postgres)) |
| Telegram | `BOT_ENV` (+ `TEST_BOT_TOKEN`/`PROD_BOT_TOKEN` or `BOT_TOKEN`), `MINI_APP_URL`, `TELEGRAM_WORKSHOP_CHAT_ID` |
| iiko | `IIKO_HOST`, `IIKO_PORT`, `IIKO_LOGIN`, `IIKO_PASSWORD` |
| Admin bootstrap | `ADMIN_USERNAME`, `ADMIN_PASSWORD` |
| Background jobs | `SYNC_INTERVAL`, `ORDER_CLEANUP_INTERVAL`, `ORDER_RETENTION`, `OUTBOX_INTERVAL` |
| Logging | `LOG_LEVEL`, `LOG_PRETTY`, `LOG_DIR` |
| Migrations | `MIGRATIONS_DIR` |

The bot token is dual-use: Telegram API access **and** the HMAC secret for
initData validation and web session tokens.

## Deployment

Railway runs one service, `bakery` (the root `Dockerfile`, config in
`railway.json`), with a volume for the SQLite file. Templates,
CSS, JavaScript and HTMX are embedded in the binary. See
[deployment.md](deployment.md) for the setup and
[frontend/ui-kit.md](frontend/ui-kit.md) for interface conventions.

## Error-handling policy

Raw DB/iiko/transport errors are **never** shown to users. Use cases return
`apperr` values with safe Russian messages; handlers log the technical cause
(with correlation ID) and respond with the safe message. The HTTP logging
middleware logs method, path, status, duration, and the error text for failed
requests.
