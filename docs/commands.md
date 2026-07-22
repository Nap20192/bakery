# Commands Cheat Sheet

How-to reference for day-to-day operations: running, restarting, code
generation, database, tests, frontend. All backend commands run from the repo
root; `.env` is loaded automatically by both binaries (`config.LoadDotenv()`),
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

## Infrastructure (Postgres + RabbitMQ)

Docker runs through **colima** on this machine — start it first:

```bash
colima start                 # start the docker VM
docker compose up -d db rabbitmq    # Postgres 17 :5432, RabbitMQ :5672 (+ UI)
docker compose ps            # check status
docker compose logs -f db    # follow logs
docker compose down          # stop everything (data volumes survive)
colima stop                  # stop the VM when done
```

Connect to the local database:

```bash
docker compose exec db psql -U "$POSTGRES_USER" -d "$POSTGRES_DB"
# or from the host:
psql "$DATABASE_URL"
```

## Run the backend

### Worker (HTTP API :8080, migrations, sync, outbox)

```bash
go run ./cmd/worker
```

Migrations apply automatically on startup; the admin user and dish catalog
are ensured/seeded on every start.

**Run in background + restart cycle** (the pattern used during development):

```bash
# build & start detached, log to /tmp
go build -o /tmp/bakery-worker ./cmd/worker && \
  nohup /tmp/bakery-worker > /tmp/bakery-worker.log 2>&1 &

# follow the log
tail -f /tmp/bakery-worker.log

# RESTART after code changes: kill old, rebuild, start again
pkill -f "/tmp/bakery-worker"; \
  go build -o /tmp/bakery-worker ./cmd/worker && \
  nohup /tmp/bakery-worker > /tmp/bakery-worker.log 2>&1 &
```

If port 8080 is busy (a stray `go run` process):

```bash
lsof -i :8080          # find the PID
kill <PID>
```

### Bot (Telegram + RabbitMQ consumer)

```bash
go run ./cmd/bot
```

Requires `BOT_ENV` + the matching `TEST_BOT_TOKEN`/`PROD_BOT_TOKEN` and
`RABBITMQ_URL` in `.env`.

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
psql "$DATABASE_URL" -c "select number, fulfillment_date, cancelled_at from orders order by created_at desc limit 10;"
```

After editing SQL in `queries/`: `make sqlc` → `make build`. Never edit
`internal/outbound/db/sqlc/` by hand — it is generated.

## Frontend (`frontend/`)

```bash
# Worker must be running on :8080.
FRONTEND_ADDR=:5173 BACKEND_URL=http://127.0.0.1:8080 go run ./frontend

# Frontend-only checks.
gofmt -w frontend/main.go frontend/internal
go test ./frontend/...
go vet ./frontend/...

# Production binary.
go build -o /tmp/bakery-frontend ./frontend
```

Browser QA is documented in
[`frontend/development-workflow.md`](frontend/development-workflow.md). Run
Playwright and axe through the installed browser skills and report the exact
tested viewports.

After changing any backend route or DTO: update `docs/api/openapi.yaml`,
run `go test ./internal/inbound/api` (route-sync test), then
`make frontend-check` (compiles the typed Go API client and renders every
template fixture).

The dev server expects the worker on :8080. Login for local web testing:
`admin` / `ADMIN_PASSWORD` from `.env`.

## Typical workflows

**Backend change:**

```bash
# 1. edit code (+ queries/*.sql → make sqlc)
make build && make vet && make test && make lint
# 2. restart the local worker (see restart cycle above)
```

**New DB field:**

```bash
# 1. add migrations/000XX_name.sql
# 2. update queries/*.sql
make sqlc
# 3. adjust repo/domain code
make build && make test
# 4. restart worker — it applies the migration on startup
```

**Fresh local environment from zero:**

```bash
colima start
docker compose up -d db rabbitmq
cp .env.example .env          # fill in tokens/passwords
go run ./cmd/worker           # applies migrations, seeds admin + catalog
# in a second terminal:
FRONTEND_ADDR=:5173 BACKEND_URL=http://127.0.0.1:8080 go run ./frontend
```

**Full reset of local data:**

```bash
docker compose down -v        # ⚠ drops the Postgres volume
docker compose up -d db rabbitmq
go run ./cmd/worker           # re-migrates and re-seeds from scratch
```

## Git conventions used in this repo

```bash
git switch development        # main working branch; PRs target main
git add -A -- . ':!cmd/testingorders/main.go'   # keep the local DSN hack out of commits
```
