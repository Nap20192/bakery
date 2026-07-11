# Database Queries Reference

Every SQL statement the application runs lives in `queries/*.sql` and is
compiled by **sqlc** into `internal/outbound/db/sqlc/` (`make sqlc` after any
edit; never edit the generated code). Only `infra/repo` adapters may call
these — sqlc types never leak past the repository layer.

Annotation meanings: `:one` returns a single row, `:many` a slice, `:exec`
nothing. `sqlc.arg` is a required parameter, `sqlc.narg` is nullable
(optional filter / NULLable column).

Tables are created by goose migrations in `migrations/` (applied by the
worker on startup).

## queries/orders.sql — orders, items, history (order service)

| Query | Kind | What it does |
|---|---|---|
| `CreateOrderCounterDay` | exec | Inserts a zero counter row for **day + shop + category** (`ON CONFLICT DO NOTHING`). Always called before `NextOrderCounter` so the increment has a row to hit. |
| `NextOrderCounter` | one | `UPDATE … SET counter = counter + 1 … RETURNING counter` — atomically claims the next order sequence number. Runs inside the create-order transaction, so concurrent creates can't duplicate numbers. |
| `CreateOrder` | one | Inserts the order header (number, location, departments, category, dates, author, comments JSON). |
| `CreateOrderItem` | one | Inserts one line: product name, optional iiko product id, quantity, reserved quantity. |
| `GetOrderByNumber` | one | Header by human-facing number (the aggregate identity). |
| `GetOrderByID` | one | Header by surrogate id (used by production re-projection). |
| `GetOrderItemsByOrderID` | many | Items of one order (заявка only), LEFT JOIN to `iiko_products` for the display `product_code`. Production facts are **not** stored here — the repo decorates items from the journal (`GetOrderProductionFacts`). |
| `GetOrderItemsByOrderIDs` | many | Same, batched via `= ANY(bigint[])` — hydrates order lists without N+1. |
| `DeleteOrderItemsByOrderID` | exec | Clears items before re-inserting on order update (full replace). |
| `CreateOrderHistory` | one | History header: who changed and when. |
| `CreateOrderHistoryItem` | one | One diff row: `change_type` (`added`/`updated`/`removed`), old/new quantity and reserve. Nullable old/new match the change type. Production writes no history. |
| `ListOrderHistoryByOrderID` | many | History headers, newest first. |
| `ListOrderHistoryItemsByHistoryID` | many | Diff rows of one history entry. |
| `UpdateOrder` | one | Updates departments, fulfillment date, comments by number. **Deliberately never touches `created_by_username`** — the author is immutable; editors live in history. |
| `ListOrders` | many | Paged list with optional filters: shop (`from_department_id`), category, exact `fulfillment_date`, or inclusive `fulfillment_from`/`fulfillment_to` range (the baker matrix loads day windows). Newest first. |
| `CountOrders` | one | Same filters as `ListOrders` — the `total` for pagination. Keep the two WHERE clauses in sync. |
| `SetOrderFavorite` | one | Toggles `is_favorite` (admin-only route). |
| `CancelOrder` | one | Soft cancel: sets `cancelled_at` + `cancelled_by_username`. |
| `RestoreOrder` | one | Clears `cancelled_at`, resets the canceller to `''`. |
| `DeleteOrdersCreatedBefore` | one | Retention cleanup: deletes orders older than the cutoff **except favorites** (`is_favorite = FALSE`), returns the deleted count. Items/history/outbox rows go via `ON DELETE CASCADE`. |

## queries/categories.sql — order categories (order service)

| Query | Kind | What it does |
|---|---|---|
| `ListOrderCategories` | many | All categories ordered by `sort_order, id`. |
| `GetOrderCategoryByID` | one | Category lookup for order create / monitor codes. |
| `CreateOrderCategory` | one | Inserts code, letter (goes into order numbers), name, color slug, sort order, `monitor_codes` array. |
| `UpdateOrderCategory` | one | Updates everything **except `code`** (the stable machine identifier); bumps `updated_at`. |
| `DeleteOrderCategory` | exec | Deletes a category; the use case first guards with `CountDishesByCategoryID`. |
| `CountDishesByCategoryID` | one | How many catalog dishes reference the category — non-zero blocks deletion (`order.category_has_dishes`). |

## queries/production.sql — production sheets / отработка (order service)

| Query | Kind | What it does |
|---|---|---|
| `CreateProductionSheet` | one | New journal document header (author; timestamps default to `now()`). |
| `GetProductionSheet` | one | Header by id. |
| `TouchProductionSheet` | exec | Bumps `updated_at` when a sheet is replaced. |
| `DeleteProductionSheet` | exec | Deletes the document; its batch rows and items cascade. |
| `ListProductionSheets` | many | Journal list: deviation count + the saved batch's order numbers per sheet, newest first. |
| `InsertProductionSheetOrder` | exec | One batch row: the sheet covers this order (saved selection, deviations optional). |
| `DeleteProductionSheetOrders` | exec | Clears a sheet's batch before re-insert (full replace on update). |
| `ListProductionSheetOrderNumbers` | many | The batch's order numbers for `GetProductionSheet`. |
| `InsertProductionSheetItem` | exec | One deviation row: sheet, order, product name, produced quantity, reason. |
| `DeleteProductionSheetItems` | exec | Clears a sheet's items before re-insert (full replace on update). |
| `ListProductionSheetItems` | many | Sheet items joined to orders for their numbers, sorted by product then order. |
| `ListProductionSheetOrderIDs` | many | Orders in a sheet's batch — the "affected set" for change detection. |
| `GetOrderProductionSheetID` | one | The sheet whose **batch** covers an order (newest wins on historical overlaps). Backs both the `production_sheet_id` in API responses and the one-sheet-per-order conflict check. |
| `GetOrderProductionFacts` | many | **The read-time decorator**: per (order, normalized product name) the newest sheet's fact (`DISTINCT ON … ORDER BY sheet_id DESC`). The repo overlays these onto order items at read time and diffs before/after journal writes to decide whether to emit `order.produced`/`production_cleared`. |

## queries/order_outbox.sql — transactional outbox (order service + relay)

| Query | Kind | What it does |
|---|---|---|
| `InsertOrderOutboxEvent` | one | Stores a domain event (aggregate id = order number, type, JSON payload, correlation id) **in the same transaction as the write** that produced it. |
| `ListUnpublishedOrderOutboxEvents` | many | Relay poll: rows with `published_at IS NULL`, oldest first, batch-limited (100). |
| `MarkOrderOutboxEventPublished` | exec | Stamps `published_at` after a successful publish. Publish-then-mark ⇒ at-least-once delivery. |

## queries/products.sql — dish catalog + iiko product lookup (order service)

| Query | Kind | What it does |
|---|---|---|
| `DishExistsByCode` | one | Does an iiko product with this code and `type = 'DISH'` exist — validates codes in bulk order text. |
| `GetIikoProductByCode` | one | Product by code, preferring `DISH` over other types, then the freshest snapshot row. |
| `GetIikoProductByID` | one | Product by iiko UUID. |
| `UpsertDishCatalogItem` | one | Catalog seed/add. On conflict by `code` updates name/theme/sort, but `category_id = COALESCE(existing, new)` — **re-seeding never clobbers an admin-assigned category**. |
| `SearchIikoDishes` | many | `ILIKE` search over DISH products by name or code (admin "add dish" picker), limited. |
| `SetDishCatalogSortOrder` | exec | One position of the admin drag-and-drop reorder. |
| `UpdateDishCatalogItem` | one | Edits a catalog row, including renaming its `code` (`new_code`). |
| `DeleteDishCatalogItem` | exec | Removes a dish from the catalog. |
| `ListDishCatalogItems` | many | Whole catalog; `sort_order = 0` (unsorted) rows sink to the end, then id/theme/name. |
| `ListDishCatalogItemsByName` | many | Case/space-insensitive name match — resolves bulk-entry lines; >1 row ⇒ `order.dish_ambiguous`. |

## queries/auth.sql — users (auth service)

| Query | Kind | What it does |
|---|---|---|
| `CreatePasswordAuthUser` | one | Creates a user; upserts by non-empty `username` (also how `EnsureAdminUser` refreshes the admin on startup). |
| `GetAuthUserByID` | one | Bearer-token resolution (web sessions). |
| `GetAuthUserByUsername` | one | Password login (returns the row incl. `password_hash`). |
| `GetAuthUserByTelegramID` | one | Bot chat resolution. |
| `GetAuthUserByTelegramUsername` | one | Mini App initData resolution (first match by id). |
| `ListAuthUsers` | many | Admin panel user list. |
| `ListAuthUsersByDepartmentID` | many | Users of a department **who have Telegram bound** (`telegram_id IS NOT NULL`) — notification recipients. |
| `ListAuthUsersByRole` | many | Same Telegram-bound filter by role — how the bot finds all bakers to DM. |
| `UpdateAuthUserRole` | one | Role change (validated against the enum in the use case). |
| `UpdateAuthUserUsername` | one | Login rename. |
| `UpdateAuthUserPassword` | one | Password **reset** (new pbkdf2 hash). |
| `BindTelegramID` | one | Attaches `telegram_id` the first time a matching-username user talks to the bot. |
| `DeleteAuthUser` | exec | Removes the account. |

## queries/departments.sql — departments (department service)

| Query | Kind | What it does |
|---|---|---|
| `GetDepartmentByID` | one | Viewer/department display, order shop resolution. |
| `GetDepartmentByCode` | one | Admin user assignment by department code. |
| `ListDepartments` | many | All departments, optionally filtered by type (`COALESCE(narg, type)` — NULL means "any"). |
| `AssignUserDepartment` | one | Sets/clears a user's `department_id` (lives here because it writes `auth_users` but is department-driven). |

## queries/iiko_snapshot.sql — sync writes (sync service)

| Query | Kind | What it does |
|---|---|---|
| `CreateIikoSyncRun` | one | Opens a sync-run record (source, date range, status `running`). |
| `FinishIikoSyncRun` | one | Closes the run with final status (`ok`/`error`), revision, error text. |
| `UpsertIikoProduct` | exec | Nomenclature row by iiko UUID: code, name, type, unit, raw JSON. |
| `UpsertIikoAssemblyChart` | exec | Assembly-chart header (assembled product, validity dates, `assembled_amount`, strategies, raw JSON). |
| `DeleteIikoAssemblyChartItemsByChartID` | exec | Clears chart items before re-insert (charts are replaced whole). |
| `InsertIikoAssemblyChartItem` | exec | Chart line with the full `amount_in/out(1-3)` matrix; **monitoring uses `amount_in`**. |
| `UpsertIikoPreparedChart` | exec | Prepared-chart header. |
| `DeleteIikoPreparedChartItemsByChartID` | exec | Same replace-whole pattern for prepared charts. |
| `InsertIikoPreparedChartItem` | exec | Prepared line (`amount`); this table — not `raw_json` — feeds prepared-chart math. |

## queries/monitor.sql — recipe-graph reads (monitor + techcard)

| Query | Kind | What it does |
|---|---|---|
| `GetActiveAssemblyChartByProductID` | one | The assembly chart valid on the order date (`date_from ≤ d ≤ date_to`, NULL `date_to` = open-ended; newest `date_from` wins). Slim projection for graph loading. |
| `GetActiveAssemblyChartFullByProductID` | one | Same selection, full row — techcard rendering. |
| `ListAssemblyChartItemsByChartID` | many | Chart lines joined to products (name/code/unit for display), sorted by `sort_weight`. Supplies `amount_in`. |
| `GetActivePreparedChartFullByProductID` | one | Prepared-chart header valid on the date. |
| `ListPreparedChartItemsByChartID` | many | Prepared lines joined to products; supplies `amount`. |

## Conventions

- **Date-valid charts**: all "active chart" lookups use the same
  `date_from ≤ order_date AND (date_to IS NULL OR date_to ≥ order_date)`
  window with `ORDER BY date_from DESC LIMIT 1`. Keep new chart queries
  identical.
- **Name matching** between orders and production/journal rows is always
  `lower(trim(name))` — product names are user-entered.
- **Full-replace updates** (order items, sheet items, chart items) delete
  then re-insert inside one transaction instead of diffing rows.
- **Optional filters** are `sqlc.narg` + `IS NULL OR column = value` so one
  query serves all filter combinations (`ListOrders`/`CountOrders` must stay
  in sync).
- Every mutation belongs to a repository transaction (`withTx`); queries
  themselves never open transactions.
- Adding a query: edit the `.sql` file → `make sqlc` → use it from the
  service's `infra/repo` only. Unused queries are dead weight — remove them
  (checked periodically with `deadcode` / grep over `Querier` methods).
