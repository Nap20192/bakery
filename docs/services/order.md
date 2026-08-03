# Order Service

`internal/services/order/` — the core of the system. Owns orders (заявки),
order categories (типы заявок), the dish catalog and order templates,
production sheets (отработка), order history, and all order domain events.

## Responsibilities

- Shop/workshop → workshop order lifecycle: create, edit, cancel, restore.
- Order numbering with per-source/per-category/per-day counters.
- Dynamic order-category dictionary (letter, color, dough monitor codes).
- Dish catalog + text templates for bulk order entry; startup seeding.
- Production sheets: the baker's record of what was actually baked.
- Order history (audit of item changes and production recalculations).
- Domain events to the transactional outbox (consumed by the bot).
- Retention cleanup of old orders.

## Layout

```
domain/            Order aggregate, OrderService (numbering, bulk parsing),
                   specifications, domain events
usecase/order/     package orderuc — UseCase + Repository ports, DTOs, service
infra/repo/        orderrepo — order_repo.go, production_repo.go (sqlc adapter)
infra/http/        orderhttp — handler.go (reads), handler_write.go,
                   handler_production.go, handler_admin.go, presenter.go
infra/outbox/      orderoutbox — outbox relay (worker only)
app/app.go         orderapp.New(queries, db) → orderuc.UseCase
```

## Domain model

### Order

Key fields (see `domain/model.go`):

| Field | Notes |
|---|---|
| `Number` | Human-facing ID and aggregate identity, e.g. `Г.Х.08.07.26.001`. |
| `Location` | Source display name, denormalized at creation. |
| `FromDepartmentID` / `ToDepartmentID` | Shop or workshop → workshop. Baker-created orders use «Цех Пекари» as the source. |
| `CategoryID` / `Category` | Order category. Required on create, **immutable on edit**. Nil on legacy orders only. |
| `CreatedByUsername` | Telegram username of the author; never overwritten by editors. |
| `Items` | Order lines. |
| `FulfillmentDate` | Date-only (UTC midnight). Defaults to created-at + 1 day. |
| `Comments` | One general comment + per-item comments (keyed by product name). |
| `Favorite` | Admin-managed flag. |
| `Cancelled`, `CancelledByUsername` | Soft cancel; cancelled orders cannot be edited until restored. |
| `ProductionSheetID` | The production sheet covering this order (nil = none). An order belongs to **at most one** sheet; a sheet covers many orders (1:N). |
| `History` | Item-level change log. |

### OrderItem

| Field | Notes |
|---|---|
| `Quantity` / `ReservedQuantity` | The заявка. Entered as `5+2` in bulk text (5 shelf + 2 reserve). |
| `ProducedQuantity *float64` | Production fact, **decorated at read time from the journal** (the order rows never store it). `nil` = "baked as ordered" — only **deviations** exist. |
| `ProducedReason` | Optional justification of the deviation («подгорело», «упало», …); decorated the same way. |
| `ProductName`, `Code` | Catalog identity. |

Derived quantities:

- `ProductionQuantity() = Quantity + ReservedQuantity` — what the workshop
  must bake.
- `EffectiveQuantity()` — `ProducedQuantity` when set, otherwise
  `ProductionQuantity()`. **Monitoring always uses this** so recorded facts
  override the original order.

### Order number

Built by `OrderService.BuildOrderNumber`:

```
<source letter>.<category letter>.<DD.MM.YY>.<NNN>
Г.Х.08.07.26.001
```

- Source letter from code/name: `gagarina`→Г, `sholokhova`→Ш,
  `saryarka`→С, `Цех Пекари`→Ц; unknown sources fall back to the first
  letter of their name.
- Category letter comes from the order category; an empty letter keeps the
  legacy format without that segment.
- `NNN` is a per **source department + category + day** counter (`order_counters`,
  day key `DDMMYYYY` from the created-at date). Counters are transactional —
  concurrent creates cannot produce duplicate numbers.

### Bulk order text (`ParseBulkOrder`)

Shop users enter orders as plain text, one item per line:

```
25.12.26                      ← optional fulfillment-date line
Сосиска в тесте 5+2           ← name + quantity (+reserve)
Кокрок с картофелем 4 // без сахара   ← "//" or ";" starts a per-line comment
```

- Lines may also start with a numeric dish code.
- Quantities must be whole numbers (`5` or `5+2`); anything else is a
  per-line validation error with a human message.
- Names are resolved against the dish catalog case-insensitively; unknown
  (`order.dish_not_found`) and ambiguous (`order.dish_ambiguous`) names are
  rejected.
- Duplicate items (same code) are rejected.
- A missing fulfillment date defaults to tomorrow (UTC).

### Domain events

Recorded on the aggregate, persisted to the outbox by the repository in the
same transaction, published by the relay (see
[architecture.md](../architecture.md#events-and-messaging)):

| Event | When |
|---|---|
| `order.created` | New order persisted. |
| `order.updated` | Items/date/comments changed. |
| `order.cancelled` / `order.restored` | Soft cancel toggled. |
| `order.produced` | The order's visible production fact changed (journal write). Carries `produced_by_username`. |
| `order.production_cleared` | All production facts removed from the order. |

Every event carries a full order snapshot.

## Production sheets (отработка)

A production sheet is a **journal document** that fixes a **batch**: the
saved selection of orders (`production_sheet_orders` — every selected order),
the loaded quantities (`production_sheet_loads` — every position), and output
deviations (`production_sheet_items`). The
journal is the **only** place the fact is stored — **an отработка never
modifies the order**. It acts as a read-time *decorator*: when the repository
hydrates an order, it overlays the journal's facts onto the items
(`GetOrderProductionFacts` + `decorateProductionFacts`, newest sheet wins per
item), which is what monitoring and the UI see as
`produced_quantity`/`produced_reason`. Dough monitoring over the sheet's
orders therefore automatically "учитывает отработку".

Rules (enforced in `usecase` + `infra/repo/production_repo.go`):

- **The batch is saved whole.** Every order of the selection lands on the
  sheet, including orders with no deviations. A sheet with zero deviations
  is valid — it pins the batch for dough calculation.
- **Loaded quantity is persisted whole.** Each item records «Закладка» even
  when it equals the order. It is operational journal data and does not alter
  the order or monitoring calculation.
- **Deviations only.** An item whose fact equals the order's
  `ProductionQuantity()` is silently dropped from the document.
- **1:N with exclusivity.** A sheet covers many orders; an order belongs to
  at most one sheet — membership is defined by the **batch selection**, not
  by deviations. Adding an order already covered by another sheet fails with
  `order.production_exists` ("edit the existing sheet").
- **The order is never modified** — neither the заявка nor any projection
  columns. Deleting a sheet simply removes the decoration; the order reads
  as "baked as ordered" again.
- **Update semantics.** `PUT /production/{id}` replaces the batch and the
  loaded quantities and deviations. Setting output back to the order quantity
  removes that deviation row; loaded quantities and the document remain until
  explicitly deleted.
- **Reasons** are optional, max 200 characters; UI offers presets
  («Подгорело», «Упало», …) plus free text.
- **No order history.** An отработка is not a change to the order, so it
  writes no `order_history` rows. `order.produced` /
  `order.production_cleared` events go to the outbox **only for orders whose
  visible fact actually changed** (before/after comparison in
  `notifyProductionChanges`) — the bot notifies with quantities and reasons.
- Cancelled orders cannot receive production facts
  (`order.production_cancelled`).

UI: creating (from the baker's selection page) and editing (from the journal)
use the **same editor** — «Заказ / Закладка / Выход», automatic proportional
distribution across orders (no manual allocation UI), optional comments opened
per product, and reason presets — plus a
«Расчёт теста» panel that runs batch monitoring over the sheet's orders.

## HTTP API

All routes are registered in `infra/http`. Errors follow the
`{"error": "<safe message>"}` envelope.

### Read/write (RequireMiniAppAuth)

| Route | Who | Notes |
|---|---|---|
| `GET /catalog` | any role | Dish catalog grouped for the editor. |
| `GET /categories` | any role | Order categories (id, code, letter, name, color, monitor codes). |
| `GET /orders` | shop sees own; baker/admin see all | Query params: `limit` (≤100, default 10), `page`, `from_department_id`, `category_id`, `fulfillment_date`, `fulfillment_from`, `fulfillment_to` (inclusive range — the baker matrix loads windows of days). |
| `POST /orders` | shop, baker, admin | Body: `{items: [{product_name, quantity, reserved_quantity}], fulfillment_date, from_department_id, category_id, comments: {general, items: [{product_name, comment}]}}`. `category_id` required; for baker the source is always «Цех Пекари» and the supplied `from_department_id` is ignored. |
| `GET /orders/{id}` | creator's shop, baker, admin | `{id}` is the order number. |
| `PUT /orders/{id}` | shop, baker, admin | Same body as create **minus** `category_id` (category never changes). |
| `POST /orders/{id}/cancel` | shop, baker, admin | Idempotent — cancelling a cancelled order returns it unchanged. |
| `POST /orders/{id}/restore` | shop, baker, admin | Idempotent for active orders. |

Roles outside `shop`/`baker`/`admin` get `403` on write routes.

### Production (RequireMiniAppAuth + baker/admin guard)

| Route | Notes |
|---|---|
| `POST /production` | Create a sheet. Body: `{orders: [{number, items: [{product_name, produced_quantity, reason}]}]}`. |
| `GET /production` | Journal list: id, author, created/updated, covered order numbers, item count. |
| `GET /production/{id}` | Sheet with items. |
| `PUT /production/{id}` | Replace items (see update semantics above). |
| `DELETE /production/{id}` | Delete + re-project affected orders. |

Other roles get `403 «Отработку ведёт пекарь или администратор.»`.

### Admin (RequireAdmin)

| Route | Notes |
|---|---|
| `GET /admin/dishes` | Catalog with codes and sort order. |
| `GET /admin/dishes/available` | iiko products searchable for adding (`code`, `name`, `unit`). |
| `POST /admin/dishes` | `{code, name, theme, category_id, sort_order}`. |
| `PUT /admin/dishes/reorder` | `{codes: [...]}` — full catalog order. |
| `PUT /admin/dishes/{code}` / `DELETE /admin/dishes/{code}` | Update/remove a dish. |
| `POST /admin/categories` | `{code, letter, name, color, sort_order, monitor_codes}`. `color` must be one of `orderdomain.CategoryColors` (`amber, sky, violet, emerald, rose, stone`). |
| `PUT /admin/categories/{id}` / `DELETE /admin/categories/{id}` | Delete fails with `order.category_has_dishes` while dishes reference the category. |
| `PATCH /orders/{id}/favorite` | `{favorite: bool}`. |

## Use-case error catalog (`orderuc`)

| Code | Kind | Meaning |
|---|---|---|
| `order.dish_not_found` | not found | Bulk line name not in catalog. |
| `order.dish_ambiguous` | conflict | Name matches several dishes. |
| `order.fulfillment_date_in_past` | invalid | Backend-enforced; the frontend past-date guard is a UX hint only. |
| `order.cancelled` | conflict | Editing a cancelled order. |
| `order.category_required` / `order.category_not_found` | invalid / not found | Create without a valid category. |
| `order.category_has_dishes` | conflict | Deleting a category still in use. |
| `order.production_order_not_found` | not found | Sheet references a missing order. |
| `order.production_sheet_not_found` | not found | Unknown sheet id. |
| `order.production_exists` | conflict | Order already covered by another sheet. |
| `order.production_cancelled` | conflict | Fact for a cancelled order. |
| `order.production_item_unknown` / `_duplicate` / `_name` / `_quantity` / `_reason_too_long` | invalid | Per-item sheet validation. |

## Background jobs (worker)

- **Cleanup ticker** — `RunCleanupTicker(interval, retention)` deletes orders
  older than `ORDER_RETENTION` every `ORDER_CLEANUP_INTERVAL`.
- **Outbox relay** — see [architecture.md](../architecture.md#events-and-messaging).
- **Catalog seeding** — `EnsureDefaultOrderTemplates` on startup parses
  `templates/dishes.txt` (→ `buns`) and `templates/bread.txt` (→ `bread`),
  upserting dishes without overwriting an admin-assigned category. Each template
  is an uppercase «группа» header followed by `<код> <название>` lines; the
  group name resolves to a `dish_groups` row (`EnsureDishGroup`).
