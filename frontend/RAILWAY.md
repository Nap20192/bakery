# Railway Frontend Setup

Frontend is a separate Railway service that serves React through a small Node HTTP server.
The browser calls the frontend public domain, and the Node server proxies `/api/*` to the backend over Railway private network.

## Services

- `frontend`: React + Node static/proxy server, public domain enabled.
- `bakery`: Go backend/bot/sync, no public domain required.
- `Postgres`: database, private network only.

## Frontend Variables

Set these on the `frontend` service:

```env
HOST=0.0.0.0
PORT=8080
BACKEND_URL=http://bakery.railway.internal:8080
```

React defaults to `/api`, so `VITE_API_BASE_URL` is not needed on Railway.
`BACKEND_URL` is used only inside the frontend container by the Node proxy.

## Backend Variables

Set this on the `bakery` service:

```env
PORT=8080
```

`HTTP_ALLOWED_ORIGINS` is not needed when the browser only calls the frontend domain and the Node server proxies to `bakery` privately.

## Frontend Service Settings

Use the repository `Nap20192/bakery`.

Recommended settings:

```text
Root Directory: /
Dockerfile Path: /frontend/Dockerfile
Config File: /frontend/railway.json
```

If deploying manually from local CLI:

```bash
railway service link frontend
railway up --detach -m "Deploy frontend"
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
