# TODO

## Reliability And Safety

- [x] Wrap order creation in a DB transaction.
  - Risk: `orders` can be created without all `order_items` if item insertion fails.
  - Code: `internal/app/order.go` (`CreateOrder`).

- [ ] Wrap order update in a DB transaction.
  - Risk: old items are deleted before new items are inserted; a failure can leave the order empty or partially updated.
  - Code: `internal/app/order.go` (`UpdateOrder`).

- [ ] Serialize iiko sync runs.
  - Risk: manual sync and background ticker can run at the same time and share the same mutable iiko client token.
  - Code: `internal/app/sync.go`, `internal/outbound/iiko/client.go`.

- [ ] Deep-copy Telegram session state before reading it outside the session mutex.
  - Risk: copied session structs share the same `items` backing array and can race with another Telegram update.
  - Code: `internal/inbound/bot/handler_order.go`, `internal/inbound/bot/session.go`.

## Access Control

- [ ] Enforce shop visibility on direct order access.
  - Risk: shop users see only their own orders in `/orders`, but can open, copy, or edit another shop's order by number.
  - Code: `internal/inbound/bot/handler_orders.go`, `internal/inbound/bot/handler_order.go`.

## Monitoring Logic

- [x] Make monitor graph truncation explicit.
  - Risk: graph loading silently stops on cycles or max depth and can undercount dough without warning.
  - Code: `internal/app/monitor.go`, `internal/domain/monitoring/service.go`.

## Performance

- [x] Reduce repeated DB graph loads in batch monitor calculations.
  - Risk: each order and each monitored ingredient reloads product graphs, causing many duplicate SQL queries.
  - Code: `internal/app/monitor.go` (`GetBatchIngredientsByCodes`, `GetIngredientsByCode`).
