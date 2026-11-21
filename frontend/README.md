# Bakery frontend

Server-rendered Telegram Mini App and web UI.

```text
frontend.go                     NewHandler: wires the API adapter into the web UI
internal/application/          CQRS ports and Queries/Commands facades
internal/backend/              API adapter (in-process transport)
internal/web/                  routes, middleware and view-model projection
internal/web/templates.go      template set, FuncMap and component helpers
internal/web/templates/        layout.html — the shell
internal/web/templates/pages/  one template per page.View, named after it
internal/web/templates/components/  partials shared between pages
internal/web/static/app.css    design tokens and responsive UI
internal/web/static/app.js     Telegram bridge and browser interactions
internal/web/static/vendor/    vendored HTMX 2.0.4
```

The API handlers and the backend adapter share JSON DTOs from
`internal/inbound/api/contract`; frontend-specific query parameters and errors
remain in `frontend/internal/application`.

The UI is served by the single binary:

```bash
go run ./cmd/bakery
```

Open <http://localhost:8080>. The health endpoint is `/health`.
See `docs/frontend/development-workflow.md` for implementation and QA rules.
