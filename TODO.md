# TODO — Architecture roadmap

Goal: consistent per-service clean architecture, no infra-type leakage, reliable
events, decoupled bot, tested critical flows.

## Architecture alignment (go-coffeeshop reference)
- [x] domain moved inside each service (internal/services/<svc>/domain)
- [x] 9. per-service `app/` composition (decentralize internal/deps god-file)
- [x] 10. HTTP handlers distributed to services (internal/services/<svc>/infra/http) + shared httpx transport

## Standardize services (ports + remove sqlc leakage)
Make every service layered like auth/order: `usecase` (UseCase/Repository
ports + DTO) + `infra/repo`. Delivery depends on interfaces, never on sqlc.

- [ ] 1. department → ports + Department DTO + infra/repo (stop leaking sqlc.Department)
- [ ] 2. techcard → ports + infra/repo
- [ ] 3. monitor → ports + infra/repo (graph/sqlc loads into repo)
- [ ] 4. sync → ports + infra/repo (+ iiko client port)

## Cross-cutting
- [x] 5. Centralized typed domain errors (internal/pkg/apperr: Kind taxonomy) + one HTTP mapper
        (httpx.WriteAppError). Sentinels carry kinds; errors.Is + KindOf both work. DTO mappers
        already live per-service (orderhttp.OrderPresenter, adminhttp.toUserResponse, departmentResponse).
- [ ] 6. Transactional outbox for order events + correlation id in envelope/logs
- [ ] 7. Bot → pure API client (add missing endpoints, http client, per-user bearer); cmd/bot drops DB
- [ ] 8. Tests: usecase (fake repos) for auth/admin/monitor/notify; repo integration (testcontainers)

## Known gaps / pre-deploy
- [ ] Dedupe duplicate telegram_username before migration 00013 (unique index)
- [ ] Verify timestamptz cast (migration 00012) against real data + backup
- [ ] Admin panel: edit telegram_username/department of existing users (currently create-only)
- [ ] monitor: cache/batch product-graph loads (N+1)
- [ ] iiko sync serialization (manual + ticker share mutable client token)
- [ ] deep-copy bot session before reading outside mutex (data race)

## Done (this effort)
- Per-service layout under internal/services; internal/app removed
- timestamptz/date types; per-shop order counter; UpdateOrder tx
- RabbitMQ event bus; bot notifications (creator + bakers + group)
- Auth: web login/password, admin-only user mgmt, bot password login + telegram_id bind
- Unique username/telegram_username/telegram_id
