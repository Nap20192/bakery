# Commands Cheat Sheet

How-to reference for day-to-day operations: running, restarting, code
generation, database, tests, frontend. All backend commands run from the repo
root; `.env` is loaded automatically by the binary (`config.LoadDotenv()`),
no `export` needed.

## Build, checks, generation (Makefile)

```bash
make build        # go build ./...
make vet          # go vet ./...
make test         # go test ./...
make lint         # golangci-lint run ./...   (must report 0 issues)
make sqlc         # sqlc generate — REQUIRED after editing any queries/*.sql
make test-orders  # ⚠ deletes ALL orders and seeds fresh test ones (dev only)
```

If the `sqlc` CLI is not installed:

```bash
go run github.com/sqlc-dev/sqlc/cmd/sqlc@latest generate
```

Full validation pass after any backend change:

```bash
make build && make vet && make test && make lint
```

Run a single test / package:

```bash
go test ./internal/services/order/... -run TestFoo
go test ./internal/services/order/usecase/order/ -v
```

## Database (SQLite)

No database server: the app opens the file at `SQLITE_PATH` (default
`data/bakery.db`, created on first start). Inspect it with the `sqlite3` CLI:

```bash
sqlite3 data/bakery.db
```

The app overrides `lower()` with a Unicode-aware version; in the CLI
`lower()` folds only ASCII, so Cyrillic case-insensitive matches differ
there.

## Run the app

One binary runs the web UI, the API, the Telegram bot and the background jobs
(iiko sync, order cleanup, outbox relay):

```bash
go run ./cmd/bakery          # http://localhost:8080
```

Migrations apply automatically on startup; the admin user and dish catalog
are ensured/seeded on every start. Without a bot token (and `BOT_ENV` other
than `prod`) the bot is disabled and web login is unavailable — set
`TEST_BOT_TOKEN` in `.env` to log in.

**Run in background + restart cycle** (the pattern used during development):

```bash
# build & start detached, log to /tmp
go build -o /tmp/bakery ./cmd/bakery && \
  nohup /tmp/bakery > /tmp/bakery.log 2>&1 &

# follow the log
tail -f /tmp/bakery.log

# RESTART after code changes: kill old, rebuild, start again
pkill -f "/tmp/bakery$"; \
  go build -o /tmp/bakery ./cmd/bakery && \
  nohup /tmp/bakery > /tmp/bakery.log 2>&1 &
```

If port 8080 is busy (a stray `go run` process):

```bash
lsof -i :8080          # find the PID
kill <PID>
```

### Health check

```bash
curl -s http://localhost:8080/health
```

## Database maintenance

```bash
# migrations run automatically, but the files live in migrations/
ls migrations/                          # NNNNN_name.sql, goose format

# reseed test orders (⚠ wipes all existing orders first)
make test-orders

# quick data peek
sqlite3 data/bakery.db "select number, fulfillment_date, cancelled_at from orders order by created_at desc limit 10;"
```

After editing SQL in `queries/`: `make sqlc` → `make build`. Never edit
`internal/outbound/db/sqlc/` by hand — it is generated.

## Frontend (`frontend/`)

The frontend is served by `go run ./cmd/bakery` (see above).

```bash
# Frontend-only checks.
gofmt -w frontend
go test ./frontend/...
go vet ./frontend/...
```

Browser QA is documented in
[`frontend/development-workflow.md`](frontend/development-workflow.md). Run
Playwright and axe through the installed browser skills and report the exact
tested viewports.

After changing any backend route or DTO: update `docs/api/openapi.yaml`,
run `go test ./internal/inbound/api` (route-sync test), then
`make frontend-check` (compiles the typed Go API client and renders every
template fixture).

Login for local web testing: `admin` / `ADMIN_PASSWORD` from `.env`.

## Typical workflows

**Backend change:**

```bash
# 1. edit code (+ queries/*.sql → make sqlc)
make build && make vet && make test && make lint
# 2. restart the local app (see restart cycle above)
```

**New DB field:**

```bash
# 1. add migrations/000XX_name.sql
# 2. update queries/*.sql
make sqlc
# 3. adjust repo/domain code
make build && make test
# 4. restart the app — it applies the migration on startup
```

**Fresh local environment from zero:**

```bash
cp .env.example .env          # fill in tokens/passwords
go run ./cmd/bakery           # applies migrations, seeds admin + catalog
```

**Full reset of local data:**

```bash
rm -f data/bakery.db*         # ⚠ deletes the local database
go run ./cmd/bakery           # re-migrates and re-seeds from scratch
```

## Git conventions used in this repo

```bash
git switch development        # main working branch; PRs target main
```
