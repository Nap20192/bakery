# Railway Frontend Setup

The frontend is a separate Go service. It renders HTML with `html/template`,
uses HTMX for partial navigation, and calls the bakery JSON API over Railway's
private network. Browsers never receive the backend bearer token: the frontend
stores the API credential in an `HttpOnly`, `SameSite=Lax` cookie.

## Services

- `frontend`: public Go HTML/HTMX service;
- `bakery`: private worker/API service;
- `Postgres` and RabbitMQ: private infrastructure.

## Frontend variables

```env
FRONTEND_ADDR=:8080
BACKEND_URL=http://bakery.railway.internal:8080
```

The container is built from the repository root because the frontend belongs
to the same Go module as the backend.

```text
Root Directory: /
Dockerfile Path: /frontend/Dockerfile
Config File: /frontend/railway.json
```

The config pins the container start command to
`/usr/local/bin/bakery-frontend`. Do not override it with `go run`: the final
Alpine image contains only the compiled binary, not the Go toolchain.

Generate a public domain only for `frontend`. Configure the worker with:

```env
MINI_APP_URL=https://<frontend-domain>
```

Verify both services through the frontend BFF:

```text
https://<frontend-domain>/health
```

Expected response: `{"status":"ok"}`. Telegram Mini App login is exchanged
for a server cookie at `POST /session/telegram`; web login uses
`POST /session/login`.
