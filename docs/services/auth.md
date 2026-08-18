# Auth Service

`internal/services/auth/` — user accounts, credentials, Telegram identity
binding, web sessions, and role management.

## Responsibilities

- User records: username, pbkdf2-hashed password, role, optional department,
  Telegram username + Telegram ID.
- Password verification for web login; HMAC session tokens.
- Telegram authentication: Mini App requests resolve the user **strictly by
  `telegram_id`** (`GetUserByTelegramID`) — the username is mutable and
  user-controlled, so it never authenticates. Binding a `telegram_id` to the
  account whose `telegram_username` matches (`AuthenticateTelegram`) is used
  **only** by the bot's `/start` flow; the Mini App binds via the `/login`
  password fallback instead.
- Admin bootstrap: the worker calls `EnsureAdminUser(ADMIN_USERNAME,
  ADMIN_PASSWORD)` on startup — creates the admin if missing, otherwise
  leaves the record untouched.
- Role definitions live in `internal/pkg/enum`; authorization is enforced by
  the HTTP middlewares and per-handler role checks (there is no separate
  permission matrix).

## Layout

```
domain/           AuthUser, role normalization (package auth / authdomain)
usecase/auth/     package authuc — UseCase + Repository ports
infra/repo/       authrepo over sqlc (users table)
infra/http/       authhttp — POST /login, GET /me
app/app.go        authapp.New(queries)
```

## Security model

- **Passwords** are hashed with `pbkdf2-sha256`; plain text is never stored
  or returned. There is **no self-registration** — the admin panel is the
  only way to create users, and password recovery is admin **reset**, never
  reveal.
- **Web sessions** (`internal/pkg/authtoken`): HMAC-signed claims
  `{user_id, expires_at}` with a **7-day TTL**, signed with the bot token as
  secret. Parsing rejects malformed tokens, bad signatures, and expired
  claims.
- **Uniqueness**: `username`, `telegram_username`, and `telegram_id` are
  unique (migration 00013 enforces the telegram username).
- The stored password hash is only surfaced through the repository's
  `GetByUsername` for verification; it never enters the domain model.

## Roles

Defined in `internal/pkg/enum` (never hardcode the strings):

| Role | Access |
|---|---|
| `admin` | Everything: user management, catalog/category admin, sync, all orders, production, monitoring. |
| `shop` | Creates/edits orders; sees only its own shop's orders. |
| `baker` | Full order CRUD, creates new orders from «Цех Пекари», runs monitoring and writes production sheets. |
| `user` | Default for a fresh record; no app access until the admin assigns a real role. |

`NormalizeRole` lowercases/trims before comparison; `IsValidRole` gates role
writes (`auth.invalid_role`).

## HTTP API

| Route | Auth | Notes |
|---|---|---|
| `POST /login` | public | `{username, password, init_data?}` → `{token, expires_at}`. Failures always answer `401 «Неверный логин или пароль.»` — no user enumeration. Returns `503` when the bot token isn't configured (the token is the HMAC secret). Optional `init_data` (raw Telegram initData) — after the password check it is HMAC-validated and the sender's `telegram_id` is bound to the account (Mini App fallback when initData didn't resolve a user, e.g. the stored `telegram_username` doesn't match). Invalid `init_data` or a bind conflict never fails the login — it is logged and skipped, the password already proved ownership. |
| `GET /me` | RequireAuth | Role, telegram identity, and the resolved department (id/code/name/type) when the user has one. Admins may legitimately have no department. |

Mini App clients normally do not call `/login` — they authenticate every
request with `Authorization: tma <initData>` (see
[architecture.md](../architecture.md#authentication)), resolved only by
`telegram_id`. The exception is the fallback above: when initData is valid
but no account has that `telegram_id` bound, the Mini App shows the password
form and sends `init_data` along with it — the password check binds the id,
and every later open logs in automatically. There is no logout in the Mini
App: the identity comes from Telegram itself, so "logging out" would just
auto-login again on the next request (see FRONTEND_BEHAVIOR.md `/me`).

## Ports

`authuc.UseCase` (used by HTTP, the bot, the admin service, and bootstrap):
user CRUD-ish operations (`CreateUserWithPassword`, `SetPassword`,
`SetUserRole`, `SetUsername`, `AssignUserDepartment`, `DeleteUser`), lookups
(`GetUserByID/TelegramID/TelegramUsername`, `ListUsers`, `ListUsersByRole`,
`ListUsersByDepartmentID`), and auth flows (`VerifyPassword`,
`AuthenticateTelegram`, `EnsureAdminUser`).

`authuc.UseCase` also exposes `BindTelegram(userID, telegramID)` — a direct
bind used by the `/login` fallback after a successful password check.

`authuc.Repository` mirrors these over the `users` table and additionally
exposes `BindTelegramID` — called when a Telegram user first talks to the bot
and their username matches an account, and by `BindTelegram`.

## Errors

| Code | Meaning |
|---|---|
| `auth.user_not_found` | Lookup miss; mapped to 403 «Пользователь не найден.» by the auth middleware (a valid Telegram login without a provisioned account). |
| `auth.invalid_role` | Role outside the enum. |
| plain `invalid credentials` | Wrong password — deliberately not an apperr so it can't leak a distinct status. |
