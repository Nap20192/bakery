# Adding Features & Keeping Frontend/Backend in Sync

How-to guide for the day-to-day development loop. The core idea: **the API
contract (`docs/api/openapi.yaml`) sits between the two sides, and two checks
keep everyone honest** — the Go route-sync test on the backend and
`tsc` typecheck of the generated client on the frontend.

```
        backend                      contract                     frontend
┌──────────────────────┐   ┌────────────────────────┐   ┌──────────────────────────┐
│ handler + RegisterRoutes │◄──│ docs/api/openapi.yaml │──►│ src/api/schema.d.ts      │
│ openapi_test.go ─ FAILS  │   │  (source of truth)    │   │ src/api/*.js (@ts-check) │
│ if code ≠ spec           │   └────────────────────────┘   │ npm run typecheck ─ FAILS│
└──────────────────────┘                                  │ if calls ≠ schema        │
                                                          └──────────────────────────┘
```

## The sync loop (memorize this)

Any time a route, request body, or response shape changes:

```bash
# 1. change the Go handler / DTO
# 2. mirror the change in docs/api/openapi.yaml
go test ./internal/inbound/api        # route-sync test must pass
# 3. regenerate + typecheck the frontend client
make api-gen                          # = cd frontend && npm run api-gen && npm run typecheck
# 4. if typecheck fails — fix the api/ facades (and only then the components)
```

Rules that make the loop work:

- **All backend calls go through `frontend/src/api/*.js`.** Components never
  call `fetch` or build URLs; one exported function = one operation from the
  spec. If a component needs a new call — add a facade function first.
- The `src/api/` layer is `// @ts-check`ed against the generated
  `schema.d.ts`. Don't remove the pragma, don't hand-edit `schema.d.ts`.
- The spec documents **reality, not plans**: never describe an endpoint that
  isn't registered (the sync test rejects it) and never register one you
  didn't describe.

## Scenario 1 — change an existing endpoint (new field, new filter)

1. Backend: add the field to the DTO in `infra/http` (presenter or request
   struct) and thread it through usecase/repo as needed. After SQL changes:
   `make sqlc`.
2. Spec: update the matching schema in `docs/api/openapi.yaml` (mark new
   response fields `required` only if the backend always sends them).
3. `make build vet test lint` + `go test ./internal/inbound/api`.
4. `make api-gen`. If the field renamed/removed, typecheck points at every
   facade and component usage to fix.
5. Update the service page in `docs/services/` if behavior changed.

## Scenario 2 — add an endpoint to an existing service

1. **Usecase**: add the method to the `UseCase` interface in
   `usecase/<svc>/interfaces.go`, implement it in the service. New data
   needs? Extend the `Repository` port + implement in `infra/repo`
   (queries → `queries/*.sql` → `make sqlc`). Update `fakeRepo` in the
   usecase tests — it must implement every port method.
2. **Handler**: add the method + route in `infra/http/handler*.go` behind the
   right middleware (`RequireMiniAppAuth` / `RequireAdmin`) and, if needed, a
   role guard inside the handler. Safe Russian error messages only.
3. **Spec**: describe the operation in `docs/api/openapi.yaml` (reuse
   components; add named schemas for new bodies).
4. **Frontend**: `make api-gen`, then add one facade function in the matching
   `src/api/` module (typed via the schema — see existing functions for the
   JSDoc pattern), then use it from the feature.
5. Validate everything (see checklist at the bottom).

## Scenario 3 — full feature end-to-end (with DB)

Order of work — inside-out on the backend, then contract, then UI:

1. **Migration**: `migrations/000XX_name.sql` (goose format, next free
   number). The worker applies it on startup — restart the local worker to
   apply.
2. **Queries**: add to `queries/<area>.sql`, run `make sqlc`. Follow the
   conventions in [database.md](database.md#conventions) (narg optional
   filters, `lower(trim())` name matching, full-replace updates).
3. **Domain**: pure logic in `internal/services/<svc>/domain` — no infra
   imports. New invariants get unit tests here.
4. **Usecase**: port methods + service logic + `apperr` sentinels with safe
   messages (`<svc>.<code>` convention). Extend `fakeRepo` in tests.
5. **Repo**: map sqlc rows → domain types. sqlc never leaks past this layer.
6. **HTTP**: handler + route + role checks.
7. **Events** (if the bot must react): domain event embedding
   `sharedkernel.Event` → record on the aggregate → repository persists to
   the outbox in the same tx → handle in `internal/inbound/bot`
   (`handler_events.go`). Names go to `internal/pkg/enum/rabbitmq.go`.
8. **Contract**: spec + `go test ./internal/inbound/api` + `make api-gen`.
9. **Frontend**: facade in `src/api/` → feature code in
   `src/features/<zone>/` composing `src/ui/` primitives (see
   [ui-kit.md](frontend/ui-kit.md)). Role-gate the UI, but remember: the
   backend is the enforcement point, frontend checks are UX.
10. **Docs**: update the service page and
    [edge-cases.md](edge-cases.md) if the feature adds domain rules.

## Scenario 4 — new service

Follow `AGENTS.md §9` (domain → usecase → repo → http → app → deps → server).
Extra contract steps: routes go into the spec like any other, and the
route-sync test picks up any `mux.Handle("METHOD /path"` in
`internal/services/<svc>/infra/http` automatically — no test changes needed.

## Scenario 5 — frontend-only feature

No contract work. Compose existing `src/api/` facades and `src/ui/`
primitives; put the code in the right `features/<zone>/`. Rules:

- `ui/` never imports from `features/`.
- New colors for categories must be added in **three** places together:
  `orderdomain.CategoryColors`, the spec's `Category.color` enum, and
  `frontend/src/lib/categories.js` (literal Tailwind classes).
- Errors are rendered via `ErrorBanner` from `ApiError.message` — never
  hand-craft messages for backend failures.

## Validation checklist (run before committing)

```bash
# backend
make build && make vet && make test && make lint    # lint must be 0 issues
# contract
go test ./internal/inbound/api                      # spec ↔ routes
# frontend
make api-gen                                        # regen + typecheck
cd frontend && npm run lint && npm run build
```

Local smoke test: restart the worker
(`pkill -f "/tmp/bakery-worker"; go build -o /tmp/bakery-worker ./cmd/worker && nohup /tmp/bakery-worker > /tmp/bakery-worker.log 2>&1 &`),
`npm run dev`, and click through the affected screens. See
[commands.md](commands.md) for the full command reference.

## Common failure modes

| Symptom | Cause / fix |
|---|---|
| `TestRoutesMatchOpenAPISpec` fails with "registered in code but absent from spec" | You added/renamed a route — describe it in `openapi.yaml`. |
| …"operations with no registered route" | Spec describes something the code doesn't serve — remove or implement. |
| `npm run typecheck` fails after `api-gen` | The contract changed under the frontend: fix the facades in `src/api/`, then follow the type errors into components. |
| `fakeRepo does not implement Repository` | You extended the port — add the method to `fakeRepo` in `order_test.go` (or the service's test file). |
| Bot doesn't notify about a new event | Event not in the outbox (record + persist in the same tx), queue/exchange name hardcoded instead of `enum/rabbitmq.go`, or no handler case in `handler_events.go`. |
| New dish/category color renders grey | Color slug missing from `lib/categories.js` (classes must be literal for Tailwind JIT). |
| Migration didn't apply | Worker wasn't restarted, or the migration number collides with an existing one. |
