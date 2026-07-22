# Domain Rules & Edge Cases

The rules below are load-bearing. Most exist because of a real scenario; a
"simplification" that drops one of them is a regression. Cross-references
point to the service pages with implementation detail.

## Orders

### Numbering & counters

- The counter key is **shop + category + day** (`DDMMYYYY` of the creation
  date). Two shops, or two categories in one shop, never share a sequence.
- Counter increment and order insert happen in one transaction — concurrent
  creates can't mint duplicate numbers.
- The order **number is the aggregate identity** (URLs, events, history);
  it never changes after creation.
- Shops without a known code letter fall back to the first letter of their
  name — a new shop still gets a stable prefix without code changes.

### Categories (типы заявок)

- A category is **required** on create and **immutable** on edit — the letter
  is baked into the number, so changing it would orphan the counter and the
  number format.
- Orders created before categories existed have `category_id = NULL`. UI and
  monitoring must handle them: `CategoryBadge` renders nothing, monitoring
  falls back to `DefaultMonitorCodes`.
- Deleting a category with dishes attached fails
  (`order.category_has_dishes`); reassign dishes first.
- A category **with no monitor codes** blocks monitoring with an explicit
  400 telling the admin to configure codes — silently computing nothing would
  look like a zero-dough day.

### Dates

- `fulfillment_date` is date-only, normalized to **UTC midnight**. Missing
  date = created-at + 1 day.
- Past-date rejection is enforced in the backend against **UTC today**
  (`order.fulfillment_date_in_past`). The frontend's guard is a UX hint only;
  never rely on it.
- Update validates against *now*, create against the order's own creation
  time — editing an old order to a past date is rejected even though creating
  it that way once succeeded.

### Bulk text entry

- Quantities must be whole; `5+2` splits into shelf + reserve
  (`ProductionQuantity = 5 + 2 = 7`).
- `//` or `;` starts a per-line comment; the earlier separator wins.
- A bare date line anywhere in the text sets the fulfillment date.
- Unknown dish names, ambiguous names, bad quantities, and unparsable lines
  produce **per-line** errors with the line number — one bad line doesn't
  discard the rest of the entry.
- Duplicate items (same code, case-insensitive name match) are rejected.
- Items resolving to a non-positive production quantity are dropped; an order
  that ends up with zero items is rejected.

### Cancel / restore

- Soft cancel only; the row is kept. **Editing a cancelled order is a
  conflict** (`order.cancelled`) — restore first.
- Cancel and restore are **idempotent**: repeating the action returns the
  order unchanged instead of erroring, so a double-tap is harmless.
- Cancelled orders cannot receive production facts.

### Visibility (RBAC)

- `shop` sees/edits **only its own shop's** orders; `baker` sees all but
  cannot create/edit shop orders; `admin` can do everything.
- Visibility failures on reads answer **404, not 403** — the API doesn't
  reveal that a foreign order number exists.
- The author (`created_by_username`) is never overwritten by later editors;
  editors are recorded in history entries only.

### Retention

- The worker deletes orders older than `ORDER_RETENTION` on
  `ORDER_CLEANUP_INTERVAL`. Anything that must survive (reports, exports)
  cannot rely on old orders being present.

## Production sheets (отработка)

- **The order is never modified.** The journal is the only store of the
  fact; `produced_quantity`/`produced_reason` in API responses are a
  **read-time decoration** overlaid by the repository (newest sheet wins per
  item, names matched via `lower(trim())`). Deleting a sheet removes the
  decoration — nothing to "undo" on the order.
- **The sheet pins the batch.** Every selected order is saved on the sheet —
  including orders without deviations — so the sheet's dough calculation
  always covers the whole batch. A sheet with zero deviations is valid and
  lives until explicitly deleted (editing values back to the заявка no longer
  deletes the document).
- **Only deviations are stored per item.** A missing fact means "baked as
  ordered"; a sheet item equal to the order quantity is silently dropped.
- **Loaded quantity («Закладка») is independent from output.** Every selected
  order item stores its loaded quantity in `production_sheet_loads`, including
  values equal to the order. Output deviations remain in
  `production_sheet_items`, so changing only the loaded quantity does not emit
  a produced/cleared event or change monitoring math.
- **One sheet per order** (order side), many orders per sheet (sheet side) —
  membership is the **batch selection**, not deviations. Touching an order
  covered by another sheet is a conflict (`order.production_exists`) — the
  fix is editing the existing sheet, never silently merging.
- Sheet items referencing products not in the order, duplicated within one
  order, with negative/NaN/Inf quantities, or with reasons over 200 chars are
  rejected item-by-item with the product name in the message.
- An отработка writes **no order history** (it is not a change to the
  order). `order.produced`/`order.production_cleared` events fire **only for
  orders whose visible fact actually changed** — re-saving identical values
  produces no notifications.
- Monitoring uses `EffectiveQuantity()`: once a fact is recorded, dough math
  follows the fact, not the plan.

## Events & notifications

- The outbox gives **at-least-once** delivery (publish-then-mark). The bot
  may receive duplicates after a crash — notifications are acceptable to
  repeat; anything stateful built on these events must be idempotent.
- Bot sends are **best-effort**: a failed DM is logged and dropped, never
  requeued (one blocked user must not wedge the queue).
- The creator is resolved by telegram username at delivery time; creators
  without a bound Telegram account are silently skipped.
- Telegram forbids WebApp buttons in group chats — the workshop group gets a
  plain URL button while DMs get a WebApp button. Don't "unify" them.

## Auth

- Mini App initData is HMAC-validated and expires after **24h**
  (`auth_date`); web bearer tokens after **7 days**. Both secrets derive from
  the bot token — rotating the bot token invalidates all sessions.
- A valid Telegram login **without a provisioned account** is 403
  («Пользователь не найден»), not 401 — the fix is admin provisioning, not
  re-login.
- Failed password login is always the same 401 message — no user
  enumeration.
- Roles outside `admin/shop/baker` (i.e. the default `user`) cannot enter the
  app at all.
- Department binding is optional (admins often have none); code that assumes
  a viewer has a department must handle its absence.

## Monitoring

- **Cycle protection**: a product already on the recursion path contributes
  0. **Depth limit 12** is an error, not a zero — a graph that deep is
  broken data.
- Zero-quantity items and unknown products contribute 0 silently (orders may
  legitimately contain items with no tech card).
- Assembly charts scale by `amount_in / assembled_amount`; prepared charts by
  `amount × qty`. The two formulas are not interchangeable.
- Prepared-chart math must read `iiko_prepared_chart_items`, not `raw_json`.
- The composition popup (`components`) expands exactly **one level** of the
  ingredient's recipe — it is a display aid, not part of the usage math.

## Catalog & seeding

- Startup seeding (`dishes.txt` → «Булочки», `bread.txt` → «Хлеб») upserts by
  code but **never overwrites a category assigned by the admin** — a redeploy
  must not undo admin work.
- Dish codes are unique; bulk entry resolves names case-insensitively, so two
  dishes differing only in case are effectively ambiguous
  (`order.dish_ambiguous`).

## Frontend

- All frontend validation is advisory; the backend re-validates everything.
- `application.Error` carries only the backend's safe message. Transport and
  non-JSON failures use a handler-owned Russian fallback.
- Inputs must stay ≥16px on phones (iOS zoom); touch targets ≥44px
  (enforced in the mobile CSS rules).
- The browser never receives a bearer token. The BFF stores API credentials
  in an `HttpOnly`, `SameSite=Lax` cookie and requires CSRF validation for
  mutations.
- Category slugs are validated by `categoryTone` before they become literal
  CSS modifier classes; unknown values render as `stone`.
