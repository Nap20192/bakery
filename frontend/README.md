# Bakery frontend

Server-rendered Telegram Mini App and web UI.

```text
main.go                         composition root and process lifecycle
internal/application/          CQRS ports and Queries/Commands facades
internal/backend/              worker HTTP API adapter
internal/web/                  routes, middleware and view-model projection
internal/web/templates/        full pages and HTMX fragments
internal/web/static/app.css    design tokens and responsive UI
internal/web/static/app.js     Telegram bridge and browser interactions
internal/web/static/vendor/    vendored HTMX 2.0.4
```

The worker handlers and the backend adapter share JSON DTOs from
`internal/inbound/api/contract`; frontend-specific query parameters and errors
remain in `frontend/internal/application`.

Run the worker API first, then:

```bash
FRONTEND_ADDR=:5173 BACKEND_URL=http://127.0.0.1:8080 go run ./frontend
```

Open <http://localhost:5173>. The default BFF health endpoint is `/health`.
See `docs/frontend/development-workflow.md` for implementation and QA rules.
