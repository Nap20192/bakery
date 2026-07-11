# Monitor Service

`internal/services/monitor/` — the dough/ingredient calculator: given an
order (or a batch of orders) and a monitored ingredient code, it computes how
much of that ingredient the workshop needs, by walking the iiko tech-card
graph.

## Layout

```
domain/           package monitoring — pure math: ProductGraph, recursion,
                  ComposeRecipe, cycle/depth protection
usecase/monitor/  package monitoruc — UseCase + Repository ports
infra/repo/       monitorrepo — loads the recipe graph from the iiko snapshot
infra/http/       monitorhttp — /monitor endpoints
app/app.go        monitorapp.New(queries)
```

The domain is pure business math; the use case loads `OrderGraph` (recipe
graph + resolved order items) through the `Repository` port and calls it.

## Calculation

Two recipe kinds, two formulas (do not change them):

- **Assembly chart** (`AssemblyRecipe`):
  `scale = ordered_quantity / assembled_amount`, each child contributes
  `item.amount_in × scale`. (`amount_in`, not `amount`.)
- **Prepared chart** (`PreparedRecipe`): each child contributes
  `item.amount × ordered_quantity`.

The recursion (`CalculateIngredientUsage`) walks the graph from each order
item's product down to the target ingredient:

- **Order quantities use `OrderItem.EffectiveQuantity()`** — the production
  fact (отработка) overrides the original заявка when recorded.
- **Cycle protection**: a product already on the current path contributes 0
  (no infinite recursion on cyclic tech cards).
- **Depth limit**: `DefaultMaxDepth = 12`; exceeding it is an error, not a
  silent 0 — it signals a pathological graph.
- Prepared-chart math reads `iiko_prepared_chart_items`, never `raw_json`.
- No caching by design.

`ComposeRecipe` additionally expands the monitored ingredient's own tech card
**one level** (the "расшифровка"/composition view), scaled to the total
usage, with the same two formulas.

## Monitor codes

Which ingredients ("dough codes") to calculate for an order comes from its
**category** (`order_categories.monitor_codes`, editable in the admin panel;
reference lists in `templates/monitor_codes.txt`). Rules:

- Order with a category → the category's codes. A category with an **empty**
  list is a configuration error: the API answers 400 telling the admin to
  configure codes.
- Legacy order without a category → `monitoring.DefaultMonitorCodes`
  (`17642, 17644, 17650, 19694`).

## Report model

```
IngredientReport
├── ingredient   IngredientUsage           total usage (code, name, unit, quantity)
├── breakdown    []IngredientDishBreakdown per order item: item qty → ingredient qty
└── components   []IngredientComponent     the ingredient's own recipe, one level,
                                           scaled to the total (ComposeRecipe)
```

Batch responses group reports per order and add `total_reports` — the same
codes summed across all requested orders.

## HTTP API (all RequireMiniAppAuth)

| Route | Notes |
|---|---|
| `GET /monitor/{id}` | Reports for order `{id}` (order number) across all of its category's monitor codes. 404 for orders the viewer cannot read (same visibility as orders). |
| `GET /monitor/{id}/{product_id}` | Report for one specific ingredient code. |
| `GET /monitor/batch?orders=N1&orders=N2…` | Multi-order calculation for the baker's selection view: per-order reports + totals. Duplicate numbers are deduped. |
| `POST /monitor/calc` | **Dough calculator**: ad-hoc calculation for manually entered items (`{category_id, items: [{code, product_name, quantity}]}`) — nothing is persisted. Codes come from the category (or the default set); tech cards valid today. Backs the `/calculator` route. |

Calculation failures (broken tech cards, missing products) answer
`400 «Не удалось посчитать калькуляцию. Проверьте заказ и техкарты.»`; the
technical cause is logged, never returned.
