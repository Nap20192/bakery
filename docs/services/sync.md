# Sync Service

`internal/services/sync/` — pulls the product catalog and tech cards from
**iiko** (the restaurant ERP) and stores a snapshot in the database. Everything
that monitoring and tech-card lookup compute is based on this snapshot.

## Layout

```
usecase/sync/   package syncuc — UseCase + ports (IikoClient, Repository)
infra/repo/     syncrepo — snapshot persistence (sync-run lifecycle +
                transactional upserts)
infra/http/     synchttp — admin-only asynchronous manual trigger
app/app.go      syncapp.New(iikoClient, db, queries, interval)
```

No domain: sync runs in the background, periodically or on an admin request.

## Ports

- **`IikoClient`** (satisfied by `internal/outbound/iiko.Client`):
  `Auth(ctx)` / `Logout(ctx)` session management,
  `ListProductsWithCategories(ctx)` — nomenclature,
  `AssemblyChartsGetAll(ctx, dateFrom, dateTo, includeDeleted, includePrepared)` —
  assembly + prepared charts.
- **`Repository`**: `SaveSnapshot(ctx, catalog, charts, syncDate)` — persists one
  fetched snapshot transactionally.

## Behavior

- `Run(ctx)` loops on `SYNC_INTERVAL` (background goroutine); `SyncOnce(ctx)`
  performs a single fetch-and-save: authenticate against iiko → fetch
  nomenclature and charts → `SaveSnapshot` → logout.
- Every iiko request uses its caller's context. Shutdown cancels in-flight
  authentication, fetches and snapshot persistence; `SyncOnce` returns the
  wrapped cancellation error and `Run` stops promptly without logging a failed
  sync at ERROR level. A failed parallel fetch also cancels its sibling.
- After successful authentication, logout always runs, including on cancellation.
  Its independent context is derived with `context.WithoutCancel` and bounded
  by a five-second timeout (`logoutTimeout`) when the sync context is not
  cancelled, allowing a remote HTTPS round trip. A cancelled sync instead uses
  a 500 ms timeout (`cancelledLogoutTimeout`) so `SyncOnce`, `Run` and manual-sync
  shutdown return in under one second even when iiko does not respond. If shutdown
  starts during logout, its remaining wait is also capped at 500 ms.
- `POST /admin/iiko/sync` returns immediately with 202 and runs with a ten-minute
  timeout tied to the app lifecycle, independently of the completed HTTP request.
  Shutdown cancels manual syncs too and waits for their independent logout to
  finish before the application exits, even if HTTP server shutdown fails.
  New triggers receive 503 during shutdown. Lifecycle cancellation is logged at
  INFO level; reaching the ten-minute manual sync deadline is logged at ERROR.
- Snapshot tables include `iiko_prepared_chart_items`, which the monitor's
  prepared-chart math reads directly (never `raw_json`).
- iiko credentials come from `IIKO_HOST`, `IIKO_PORT`, `IIKO_LOGIN`,
  `IIKO_PASSWORD`; the app fails to start when they are missing
  (`WithIikoClient` validates).
- iiko integration tests skip automatically when those env vars are unset;
  recorded results live in `internal/outbound/iiko/testdata/results`.

## Consumers of the snapshot

- **monitor** — recipe graph for ingredient calculation.
- **techcard** — tech-card lookup by product code.
- **order admin** — `GET /admin/dishes/available` searches iiko products when
  adding dishes to the catalog.
