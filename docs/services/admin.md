# Admin Service

`internal/services/admin/` — the backend of the admin panel's user
management. It is a thin **composition service**: it owns no tables and no
domain of its own, but orchestrates the auth and department services behind
ports.

## Layout

```
usecase/admin/    package adminuc — UseCase + ports (UserAccounts, Departments) + DTOs
infra/http/       adminhttp — /users* and /admin/departments routes
app/app.go        adminapp.New(authService, departmentService)
```

There is no `domain/` or `infra/repo/`: persistence happens inside the auth
and department services. The composition root wires
`adminapp.New(deps.AuthService, deps.DepartmentService)` — which is why
`WithAdminService()` must come after both in `internal/deps`.

## Ports

- **`UserAccounts`** — satisfied by the auth service: create with password,
  list, set role/username/password, assign department, delete.
- **`Departments`** — satisfied by a small adapter over the department
  service: `GetByCode`, `ListAll`.

The use case translates between transport-level DTOs (`adminuc.User`,
`adminuc.CreateUserInput` with a `DepartmentCode`) and the auth domain
(department **ID**), resolving codes through the `Departments` port.

## HTTP API (all RequireAdmin)

| Route | Notes |
|---|---|
| `GET /users` | All users with role, telegram identity, department. |
| `POST /users` | `{username, password, telegram_username, role, department_code}`. Role must pass `IsValidRole`; department is optional. |
| `PATCH /users/{id}` | Partial update: any of role / username / password / department code. Password change is a **reset** — the current password is never shown. |
| `DELETE /users/{id}` | Remove the account. |
| `GET /admin/departments` | All departments for the assignment dropdown. |

## Invariants

- No self-registration anywhere in the system; this API is the only way to
  create users.
- Passwords are write-only (pbkdf2 handled by the auth service — see
  [auth.md](auth.md#security-model)).
- Role strings come from `internal/pkg/enum`; invalid roles are rejected with
  `auth.invalid_role`.
