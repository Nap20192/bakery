# Department Service

`internal/services/department/` — the dictionary of physical locations:
shops (магазины) and the workshop (цех).

## Layout

```
usecase/department/   package departmentuc — UseCase + Repository + Department DTO
infra/repo/           departmentrepo over sqlc (departments table)
infra/http/           departmenthttp — GET /departments
app/app.go            departmentapp.New(queries)
```

No domain package: departments are reference data with no behavior.

## Model

```go
type Department struct {
    ID   int64
    Code string // stable machine code: gagarina, saryarka, sholokhova, workshop…
    Name string // display name: «Магазин Гагарина», «Цех Пекари»
    Type string // enum.DepartmentType: "shop" | "workshop"
}
```

Default departments are seeded by migration 00003: «Магазин Гагарина»,
«Магазин Сарыарка», «Магазин Шолохова», «Цех Пекари».

## Behavior and consumers

- `ListByType(shop|workshop)`, `GetByCode`, `GetByID`.
- **Order flow** depends on types: orders go shop/workshop → workshop; the
  source code/name feeds the order-number prefix (Г/С/Ш/Ц — see
  [order.md](order.md#order-number)).
- **Auth middleware** resolves the viewer's optional department for display
  and for shop-scoped order filtering.
- **Admin service** resolves department codes to IDs when assigning users.

## HTTP API

| Route | Auth | Notes |
|---|---|---|
| `GET /departments?type=shop\|workshop` | RequireMiniAppAuth | Departments of the given type. Used by the order editor's from-department picker; every authenticated viewer gets the full list — the shop is chosen at order time, not bound to the user. |
| `GET /admin/departments` | RequireAdmin | Registered by the **admin** service; full list for user assignment. |
