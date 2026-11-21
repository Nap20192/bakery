# Deployment (Railway)

The whole application is one container built from the root `Dockerfile`; the
Railway config is `railway.json` in the repository root. Railway runs two
services:

- `bakery` — public; serves the web UI and the Telegram Mini App, runs the
  Telegram bot and the background jobs;
- PostgreSQL — private.

The JSON API is not exposed on any port: the web UI calls it inside the
process. Browsers never receive the API bearer token: the web UI stores it in
an `HttpOnly`, `SameSite=Lax` cookie.

## Service settings

```text
Root Directory: /
Config File:    /railway.json
```

`railway.json` pins the start command to `/app/bakery` and the health check to
`/health`. Do not override the start command with `go run`: the runtime image
contains only the compiled binary, migrations and seed templates.

Generate a public domain for `bakery` and set `MINI_APP_URL` to it.

## Variables

```env
DATABASE_URL=${{Postgres.DATABASE_URL}}
BOT_ENV=prod
PROD_BOT_TOKEN=...
MINI_APP_URL=https://<bakery-domain>
TELEGRAM_WORKSHOP_CHAT_ID=...
ADMIN_PASSWORD=...
IIKO_HOST=...
IIKO_LOGIN=...
IIKO_PASSWORD=...
```

Railway sets `PORT`; the binary listens on it. The process refuses to start
without the bot token for its `BOT_ENV` or with an invalid `MINI_APP_URL`, and
exits when Telegram rejects the token.

## One replica only

Keep `numReplicas` at 1. Telegram allows one long-polling consumer per bot
token: a second replica (or a leftover bot service using the same token) makes
`getUpdates` fail with a conflict.

## Migrating from the three-service setup

The previous setup ran `frontend`, `bakery` (worker), `bot` and RabbitMQ as
separate services. To switch:

1. Check that the old bot has caught up: the `bakery.bot.order-events` queue
   in the RabbitMQ management UI shows 0 messages ready and 0 unacked. Events
   still sitting in the queue were marked published long before the cutover
   and the replay below would not cover them. Then note the cutover start time
   (UTC).
2. Point the `bakery` service at `/railway.json` and give it the public
   domain (move it from `frontend`).
3. Copy the bot variables (`BOT_ENV`, `PROD_BOT_TOKEN`,
   `TELEGRAM_WORKSHOP_CHAT_ID`) to `bakery` and set `MINI_APP_URL` to its
   domain. If the domain changed, update the Mini App / menu button URL in
   BotFather as well, or the Mini App keeps opening the deleted service.
4. Delete the `bot` service before the new deploy goes live, so two pollers
   never share the token.
5. Once `/health` on the new deploy answers, replay the events the old
   worker relayed into RabbitMQ after the bot was gone (they are marked
   published but nobody consumed them):

   ```sql
   UPDATE order_outbox SET published_at = NULL
   WHERE published_at >= '<cutover start>';
   ```

   Delivery is at-least-once, so an event notified twice is acceptable.
6. Delete the `frontend` and RabbitMQ services.
7. Remove the obsolete variables: `RABBITMQ_URL`, `BACKEND_URL`,
   `FRONTEND_ADDR`, `HTTP_ALLOWED_ORIGINS`.

## Verify

```text
https://<bakery-domain>/health
```

Expected response: `{"status":"ok"}` — it goes through the web UI into the
in-process API; it does not check the database. Telegram Mini App login is exchanged for a server cookie at
`POST /session/telegram`; web login uses `POST /session/login`.
