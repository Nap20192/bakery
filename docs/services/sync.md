# Sync Service

`internal/services/sync/` — pulls the product catalog and tech cards from
**iiko** (the restaurant ERP) and stores a snapshot in Postgres. Everything
that monitoring and tech-card lookup compute is based on this snapshot.

## Layout

```
usecase/sync/   package syncuc — UseCase + ports (IikoClient, Repository)
infra/repo/     syncrepo — snapshot persistence (sync-run lifecycle +
                transactional upserts)
app/app.go      syncapp.New(iikoClient, db, queries, interval)
```

No domain and no HTTP: sync is a background capability of the worker.

## Ports

- **`IikoClient`** (satisfied by `internal/outbound/iiko.Client`):
  `Auth()` / `Logout()` session management,
  `ListProductsWithCategories()` — nomenclature,
  `AssemblyChartsGetAll(dateFrom, dateTo, includeDeleted, includePrepared)` —
  assembly + prepared charts.
- **`Repository`**: `SaveSnapshot(catalog, charts, syncDate)` — persists one
  fetched snapshot transactionally.

## Behavior

- `Run(ctx)` loops on `SYNC_INTERVAL` (worker goroutine); `SyncOnce(ctx)`
  performs a single fetch-and-save: authenticate against iiko → fetch
  nomenclature and charts → `SaveSnapshot` → logout.
- Snapshot tables include `iiko_prepared_chart_items`, which the monitor's
  prepared-chart math reads directly (never `raw_json`).
- iiko credentials come from `IIKO_HOST`, `IIKO_PORT`, `IIKO_LOGIN`,
  `IIKO_PASSWORD`; the worker fails to start when they are missing
  (`WithIikoClient` validates).
- iiko integration tests skip automatically when those env vars are unset;
  recorded results live in `internal/outbound/iiko/testdata/results`.

## Consumers of the snapshot

- **monitor** — recipe graph for ingredient calculation.
- **techcard** — tech-card lookup by product code.
- **order admin** — `GET /admin/dishes/available` searches iiko products when
  adding dishes to the catalog.
