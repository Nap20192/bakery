# Railway Frontend Setup

Frontend is a separate Railway service that serves React through nginx.
The browser calls the frontend public domain, and nginx proxies `/api/*` to the backend over Railway private network.

## Services

- `frontend`: React + nginx, public domain enabled.
- `bakery`: Go backend/bot/sync, no public domain required.
- `Postgres`: database, private network only.

## Frontend Variables

Set these on the `frontend` service:

```env
PORT=8080
VITE_API_BASE_URL=/api
BACKEND_URL=http://bakery.railway.internal:8080
```

`VITE_API_BASE_URL=/api` makes the React app call its own domain.
`BACKEND_URL` is used only inside the frontend container by nginx.

## Backend Variables

Set this on the `bakery` service:

```env
PORT=8080
```

`HTTP_ALLOWED_ORIGINS` is not needed when the browser only calls the frontend domain and nginx proxies to `bakery` privately.

## Frontend Service Settings

Use the repository `Nap20192/bakery`.

Recommended settings:

```text
Root Directory: /frontend
Dockerfile Path: /Dockerfile
Config File: /railway.json
```

If deploying manually from local CLI:

```bash
railway service link frontend
railway up ./frontend --path-as-root --detach -m "Deploy frontend"
```

## Public Domains

Generate a public domain only for `frontend`.

Remove the public domain from `bakery` after verifying:

```text
https://<frontend-domain>/api/health
```

Expected response:

```json
{"status":"ok"}
```
