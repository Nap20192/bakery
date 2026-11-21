# Deployment (Railway)

The whole application is one container built from the root `Dockerfile`; the
Railway config is `railway.json` in the repository root. Railway runs one
service, `bakery`, with a volume that holds the SQLite database. It serves the
web UI and the Telegram Mini App, runs the Telegram bot and the background
jobs.

The JSON API is not exposed on any port: the web UI calls it inside the
process. Browsers never receive the API bearer token: the web UI stores it in
an `HttpOnly`, `SameSite=Lax` cookie.

## Service settings

```text
Root Directory: /
Config File:    /railway.json
Volume:         mounted at /data
```

`railway.json` pins the start command to `/app/bakery` and the health check to
`/health`. Do not override the start command with `go run`: the runtime image
contains only the compiled binary, migrations and seed templates.

Generate a public domain for `bakery` and set `MINI_APP_URL` to it.

## Variables

```env
SQLITE_PATH=/data/bakery.db
RAILWAY_RUN_UID=0
BOT_ENV=prod
PROD_BOT_TOKEN=...
MINI_APP_URL=https://<bakery-domain>
TELEGRAM_WORKSHOP_CHAT_ID=...
ADMIN_PASSWORD=...
IIKO_HOST=...
IIKO_LOGIN=...
IIKO_PASSWORD=...
```

- Railway mounts volumes as `root`, while the image runs as an unprivileged
  user; `RAILWAY_RUN_UID=0` lets the process write to the volume.
- Railway sets `PORT`; the binary listens on it.
- The process refuses to start without the bot token for its `BOT_ENV` or
  with an invalid `MINI_APP_URL`, and exits when Telegram rejects the token.

## One replica, short downtime on deploy

Keep `numReplicas` at 1. Railway does not allow replicas on a service with a
volume, and Telegram allows one long-polling consumer per bot token. Railway
stops the old deployment before the new one mounts the volume, so every deploy
has a short downtime even with the health check.

## Backups

The database is one file on the volume, so backing it up is a volume backup.
In the service's **Backups** tab enable a schedule — Daily (kept 6 days) and
Weekly (kept 27 days) are a sensible pair. Backups are billed as incremental
volume storage. Restoring creates a new volume from the backup and applies on
redeploy; backups can only be restored into the same project and environment,
and wiping the volume deletes its backups.

## Moving from Postgres

Earlier versions stored data in a Railway PostgreSQL service. The first start
with `IMPORT_POSTGRES_URL` set and no database file imports everything from
Postgres, including undelivered order events, then starts normally. The
import writes to a temporary file and moves it into place only after every
table copied with matching row counts and foreign keys checked; on any error
the process exits and leaves no database file, so the next start retries.

1. Attach the volume at `/data` and let that deploy finish. From now on Railway
   never runs two deployments of the service at once, so no write can land in
   Postgres after the import has read it.
2. Set `SQLITE_PATH=/data/bakery.db`, `RAILWAY_RUN_UID=0` and
   `IMPORT_POSTGRES_URL=${{Postgres.DATABASE_URL}}`, then deploy the SQLite
   version.
3. Check the log for `postgres import complete` and open `/health`, the orders
   list and the production journal.
4. Remove `IMPORT_POSTGRES_URL`. With the variable left in place, a wiped volume
   would silently re-import stale Postgres data.
5. Keep the Postgres service until you have checked the data, then delete it.

The same import runs locally with
`go run ./cmd/pg2sqlite -from <postgres-url> -to <file>`.

## Moving from the three-service setup

The setup before that ran `frontend`, `bakery` (worker), `bot` and RabbitMQ as
separate services on Postgres. Switch to the single binary on Postgres first,
then move to SQLite as above:

1. Note the cutover start time (UTC).
2. Point the `bakery` service at `/railway.json` and give it the public
   domain (move it from `frontend`).
3. Copy the bot variables (`BOT_ENV`, `PROD_BOT_TOKEN`,
   `TELEGRAM_WORKSHOP_CHAT_ID`) to `bakery` and set `MINI_APP_URL` to its
   domain. If the domain changed, update the Mini App / menu button URL in
   BotFather as well, or the Mini App keeps opening the deleted service.
4. Delete the `bot` service before the new deploy goes live, so two pollers
   never share the token.
5. Once `/health` on the new deploy answers, replay in Postgres the events the
   old worker relayed into RabbitMQ after the bot was gone (they are marked
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
in-process API; it does not check the database. Telegram Mini App login is
exchanged for a server cookie at `POST /session/telegram`; web login uses
`POST /session/login`.
