# Adding Features & Keeping Frontend/Backend in Sync

`docs/api/openapi.yaml` is the boundary between the worker API and the Go
HTML/HTMX BFF.

```text
worker handler <-> shared Go DTOs <-> backend adapter <-> Query/Command <-> web
route-sync test    OpenAPI contract       Go compiler + template smoke tests
```

## Contract loop

After changing a route, request body, or response:

```bash
# update backend handler / DTO and docs/api/openapi.yaml
go test ./internal/inbound/api

# update shared contract, backend adapter and Query/Command gateway
make frontend-check
```

The backend adapter never exposes raw worker errors. `application.Error`
contains the safe API message; web handlers own the fallback for transport
and malformed responses.

## Existing service endpoint

1. Extend the use-case interface and implementation; extend its repository
   port and adapter if data access changes.
2. Update fake repositories and domain/use-case tests.
3. Add the HTTP handler and route behind the correct auth middleware.
4. Update `docs/api/openapi.yaml` and run the route-sync test.
5. Add/update the shared DTO in `internal/inbound/api/contract`, the typed
   method in `frontend/internal/backend`, and its application Query or Command
   gateway.
6. Add the BFF route, handler, view model, and template under
   `frontend/internal/web`. Business rules remain in the worker; BFF
   validation is only ergonomic.
7. Run `make frontend-check` and the full validation loop.

## Full feature with database changes

1. Add a goose migration in `migrations/`.
2. Add SQL in `queries/` and run `make sqlc`.
3. Implement pure domain rules and tests.
4. Implement use case, repository adapter, safe `apperr`, and tests.
5. Add transport handler and role checks.
6. Add outbox event persistence and bot handling when notification behavior
   changes.
7. Update OpenAPI and its route-sync test.
8. Update the shared contract, backend adapter, Query/Command gateway, web
   handler, templates, CSS/JS, and template fixtures.
9. Update service, edge-case, and frontend behavior documentation.

## Frontend-only feature

Use the smallest complete vertical slice:

```text
route -> handler -> Query/Command -> typed view model -> template -> CSS/JS enhancement
```

- Full pages render through `layout`; reusable partials are named templates.
- Browser API calls are forbidden except the Telegram session exchange to
  the same BFF. Worker reads and writes belong behind application Queries and
  Commands, implemented by `frontend/internal/backend`.
- Use semantic CSS tokens and existing structural classes before adding new
  primitives.
- HTMX attributes express requests and swaps. Plain JavaScript is reserved for
  local interaction state that hypermedia cannot represent cleanly.
- Native HTML controls (`dialog`, `details`, forms, tables) are preferred.
- Keep category slug validation in sync across domain, OpenAPI,
  `categoryTone`, and literal CSS modifiers.

## Validation

```bash
make build && make vet && make test && make lint
go test ./internal/inbound/api
make frontend-check
git diff --check
```

For visible work, run the real worker and frontend and verify 375x812,
768x1024, 1280x800, and 1440x900 with keyboard, console/network inspection,
and axe. See [frontend/development-workflow.md](frontend/development-workflow.md).

## Common failures

| Symptom | Cause / fix |
|---|---|
| Route-sync test reports missing operation | Update `docs/api/openapi.yaml` or register the route. |
| Backend adapter decode fails | Shared DTO no longer matches OpenAPI or the worker response; update `internal/inbound/api/contract`. |
| Template smoke test fails | Template field/function no longer matches its view model. |
| HTMX error response is blank | Return safe HTML and keep the `htmx:beforeSwap` error handling. |
| Mutation returns 403 before API call | CSRF cookie/form token is missing or stale; render through `layout`. |
| Category renders neutral | Add the slug to domain/spec/helper/CSS together. |
