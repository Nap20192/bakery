# Bakery — план устранения аудита

Этот файл планирует устранение дефектов из [`bugs.md`](bugs.md). Существующий
[`TODO.md`](TODO.md) — отдельный архитектурный roadmap и не заменяется этим
документом.

Правила выполнения:

- один checkbox закрывается только вместе с тестом, который падал до исправления;
- сначала исправляется корневая инварианта в БД/domain/usecase, затем transport/UI;
- после SQL — `make sqlc`; после HTTP contract — обновление
  `docs/api/openapi.yaml` и contract tests;
- не смешивать P0 data fixes с визуальными рефакторами;
- после каждого блока рабочее дерево должно проходить указанные проверки.

## Этап 0 — зафиксировать воспроизведения

- [ ] **T00.1** Добавить PostgreSQL integration-тест гонки двух отработок
  ([BUG-001](bugs.md#bug-001--один-заказ-может-попасть-в-две-отработки)).
- [ ] **T00.2** Добавить тест, что повторный admin-create не меняет существующего
  пользователя ([BUG-002](bugs.md#bug-002--создание-пользователя-перезаписывает-существующий-аккаунт)).
- [ ] **T00.3** Добавить конкурентный cancel/update test
  ([BUG-004](bugs.md#bug-004--отменённый-заказ-можно-изменить-в-гонке)).
- [ ] **T00.4** Зафиксировать тестами текущие production snapshots, mixed category,
  incomplete items и post-production edit
  ([BUG-005](bugs.md#bug-005--редактирование-заказа-ломает-сохранённую-отработку),
  [BUG-012](bugs.md#bug-012--отработка-допускает-заказ-без-единой-позиции),
  [BUG-014](bugs.md#bug-014--смешанную-категорию-можно-сохранить-в-одной-отработке)).

Проверка этапа:

```bash
go test ./internal/services/order/... ./internal/services/auth/... ./internal/services/admin/...
```

## Этап 1 — P0 database и account invariants

- [ ] **T01.1** Добавить миграцию `UNIQUE(order_id)` для
  `production_sheet_orders`; перед созданием constraint проверить и явно
  обработать исторические дубли ([BUG-001](bugs.md#bug-001--один-заказ-может-попасть-в-две-отработки)).
- [ ] **T01.2** Убрать общий `ON CONFLICT DO NOTHING` из membership insert,
  преобразовать unique violation в `409 order.production_exists`.
- [ ] **T01.3** Разделить «создать пользователя» и idempotent «ensure admin»;
  обычный create должен быть чистым INSERT и возвращать `409`
  ([BUG-002](bugs.md#bug-002--создание-пользователя-перезаписывает-существующий-аккаунт)).
- [ ] **T01.4** Добавить уникальность нормализованной буквы категории и
  application validation с безопасным сообщением
  ([BUG-003](bugs.md#bug-003--одинаковые-буквы-категорий-блокируют-создание-заказов)).
- [ ] **T01.5** Сделать update/cancel/production state checks атомарными в SQL
  или под row lock; не публиковать updated/produced после cancel
  ([BUG-004](bugs.md#bug-004--отменённый-заказ-можно-изменить-в-гонке)).

Проверка этапа:

```bash
make sqlc
go test ./internal/services/order/... ./internal/services/auth/... ./internal/services/admin/...
go test ./internal/inbound/api
```

## Этап 2 — production ledger и category correctness

- [ ] **T02.1** Выбрать постоянную модель неизменяемой идентичности позиции
  production sheet. Предпочтительно хранить snapshot/stable product identity;
  временный запрет смены состава уже отработанного заказа допустим отдельным
  коротким guard ([BUG-005](bugs.md#bug-005--редактирование-заказа-ломает-сохранённую-отработку)).
- [ ] **T02.2** Убрать inner join зависимости чтения листа от текущего
  `order_items`; мигрировать/проверить существующие orphan loads.
- [ ] **T02.3** В общей validation create/update/draft сверять category блюда с
  category заказа, сохранив правило uncategorized dish
  ([BUG-006](bugs.md#bug-006--категория-заказа-не-сверяется-с-категорией-блюда)).
- [ ] **T02.4** Требовать полный набор уникальных позиций каждого заказа в
  production input ([BUG-012](bugs.md#bug-012--отработка-допускает-заказ-без-единой-позиции)).
- [ ] **T02.5** Отклонять mixed-category production до любых записей
  ([BUG-014](bugs.md#bug-014--смешанную-категорию-можно-сохранить-в-одной-отработке)).
- [ ] **T02.6** Применять workshop-source rule к baker update
  ([BUG-015](bugs.md#bug-015--пекарь-может-сменить-источник-своего-заказа-на-магазин)).
- [ ] **T02.7** Исправить стабильный ключ history diff при пустом iiko code
  ([BUG-019](bugs.md#bug-019--отсутствие-iiko-row-создаёт-ложную-историю-removedadded)).

Проверка этапа:

```bash
make sqlc
go test ./internal/services/order/...
go test -race ./internal/services/order/...
```

## Этап 3 — frontend correctness

- [ ] **T03.1** Исправить linked-state сохранённой отработки: связь определяется
  равенством output/load, не output/order
  ([BUG-007](bugs.md#bug-007--редактор-отработки-незаметно-сохраняет-старый-выход)).
- [ ] **T03.2** Переписать распределение в целых десятых без отрицательного
  остатка; проверить суммы и zero total
  ([BUG-013](bugs.md#bug-013--пропорциональное-округление-может-создать-отрицательную-долю)).
- [ ] **T03.3** Загружать все страницы матрицы/таблицы и передавать category
  worker-side ([BUG-008](bugs.md#bug-008--матрица-и-таблица-молча-обрезают-заказы-после-первых-100)).
- [ ] **T03.4** Ограничить/нормализовать date window до построения матрицы
  ([BUG-009](bugs.md#bug-009--произвольный-диапазон-дат-может-исчерпать-память)).
- [ ] **T03.5** Привязать batch selection к viewer и очищать при смене сессии
  ([BUG-010](bugs.md#bug-010--выбранная-партия-протекает-между-аккаунтами)).
- [ ] **T03.6** Добавить «Испечено», delta/reason и role-correct sheet link в
  order detail ([BUG-026](bugs.md#bug-026--detail-заказа-не-показывает-обязательный-факт-выпечки)).
- [ ] **T03.7** Закрыть stale state: calculator result, `category_id=0`,
  pagination filters, table start и draft failures
  ([BUG-027](bugs.md#bug-027--frontend-теряет-или-показывает-устаревшее-состояние)).
- [ ] **T03.8** Сделать user PATCH атомарным
  ([BUG-016](bugs.md#bug-016--patch-пользователя-частично-коммитит-неуспешный-запрос)).
- [ ] **T03.9** Определить code категории как mutable или immutable и привести
  SQL, DTO, форму и spec к одному контракту
  ([BUG-017](bugs.md#bug-017--изменение-code-категории-возвращает-ложный-успех)).
- [ ] **T03.10** Обрезать monitor cycle в repository без ошибки, сохранив
  max depth ([BUG-018](bugs.md#bug-018--цикл-техкарт-превращается-в-ошибку-до-доменного-расчёта)).

Проверка этапа:

```bash
gofmt -w <изменённые-go-файлы>
go test ./frontend/... ./internal/services/admin/... ./internal/services/monitor/...
go vet ./frontend/... ./internal/services/admin/... ./internal/services/monitor/...
make frontend-check
```

## Этап 4 — API projection и производительность

- [ ] **T04.1** Ограничить selection максимумом до выполнения per-order calls
  ([BUG-011](bugs.md#bug-011--страницы-партии-делают-до-сотен-последовательных-http-вызовов)).
- [ ] **T04.2** Добавить bulk projection выбранных заказов/production detail,
  чтобы BFF не выполнял последовательный lookup для каждого номера.
- [ ] **T04.3** Добавить пагинацию production journal и category projection
  прямо в list response; не проглатывать lookup errors.
- [ ] **T04.4** Обновить OpenAPI, shared contract, backend adapter, fixtures и
  route/spec tests в одном vertical slice.
- [ ] **T04.5** Убрать повторный `/me` при backend error
  ([BUG-023](bugs.md#bug-023--ошибка-backend-вызывает-повторный-me-и-удваивает-timeout)).
- [ ] **T04.6** Сделать reorder single-flight/idempotent
  ([BUG-024](bugs.md#bug-024--reorder-допускает-повторную-отправку)).

Проверка этапа:

```bash
go test ./internal/inbound/api ./internal/services/order/... ./frontend/...
make frontend-check
```

Дополнительно измерить число BFF→worker запросов:

- selection из 100 заказов — O(1), не 101;
- production detail из 100 заказов — O(1), не 102;
- страница журнала — фиксированное число запросов на страницу.

## Этап 5 — auth и доставка frontend assets

- [ ] **T05.1** Добавить CSRF/Origin protection login и Telegram exchange,
  ротировать CSRF после успешной аутентификации
  ([BUG-020](bugs.md#bug-020--loginsession-exchange-освобождён-от-csrf-и-не-throttled)).
- [ ] **T05.2** Добавить и документировать rate limit на login в приложении или
  edge; regression проверяет `429` на burst.
- [ ] **T05.3** Загружать Telegram bridge неблокирующе после local HTMX/app.js
  и проверить обычный web login и TMA login
  ([BUG-021](bugs.md#bug-021--telegram-script-блокирует-htmx-и-appjs)).
- [ ] **T05.4** Добавить versioned static URLs, cache policy и compression либо
  зафиксировать проверенную ответственность reverse proxy
  ([BUG-022](bugs.md#bug-022--static-assets-отдаются-без-cache-policy-и-compression)).

Проверка этапа:

```bash
go test ./frontend/...
make frontend-check
```

Browser/network acceptance:

- stalled/blocked Telegram CDN не блокирует local UI;
- повторная загрузка static даёт cache hit/304;
- production response с `Accept-Encoding` сжат;
- hostile-origin tokenless auth отклонён;
- burst неверных login ограничен.

## Этап 6 — mobile UX и accessibility

- [ ] **T06.1** Убрать overflow sticky submit на 320/375 px; кнопки могут
  переноситься/становиться сеткой, но остаются минимум 44 px
  ([BUG-025](bugs.md#bug-025--форма-нового-заказа-шире-мобильного-viewport)).
- [ ] **T06.2** Исправить primary/focus/input contrast до WCAG AA
  ([BUG-028](bugs.md#bug-028--набор-accessibility-дефектов)).
- [ ] **T06.3** Увеличить preview/comment/view tab targets до 44×44.
- [ ] **T06.4** Убрать nested interactive controls из selection card, сохранив
  keyboard selection и обычную навигацию.
- [ ] **T06.5** Добавить skip link, имя confirm dialog и правильный focus
  indicator для обеих sheet tabs.
- [ ] **T06.6** Исправить login semantic hierarchy (`h1`) и пустую desktop-колонку.
- [ ] **T06.7** Выполнить полный keyboard traversal и automated Axe scan.

Обязательные viewport:

- 375×812;
- 768×1024;
- 1280×800;
- 1440×900.

На каждом проверить основной flow, keyboard, console, network, document-level
overflow и horizontal scrollers.

## Этап 7 — tooling и финальная регрессия

- [ ] **T07.1** Вернуть `frontend/` в scope golangci-lint
  ([BUG-029](bugs.md#bug-029--lint-не-проверяет-frontend)).
- [ ] **T07.2** Устранить G101 dummy DSN узким test-only исправлением или
  обоснованным suppression.
- [ ] **T07.3** Обновить [`docs/edge-cases.md`](docs/edge-cases.md),
  [`docs/constraints.md`](docs/constraints.md), service docs и
  [`frontend/FRONTEND_BEHAVIOR.md`](frontend/FRONTEND_BEHAVIOR.md), если
  окончательное решение меняет контракт или ограничение.
- [ ] **T07.4** После закрытия каждого BUG отметить его как закрытый в
  [`bugs.md`](bugs.md) с ссылкой на regression test/commit.

Финальный gate:

```bash
make build
make vet
make test
make lint
go test ./internal/inbound/api
make frontend-check
```

Дополнительно:

```bash
go test -race ./frontend/... ./internal/services/order/... ./internal/services/monitor/...
git diff --check
```

## Рекомендуемый порядок PR

1. **DB exclusivity + user create semantics** — BUG-001, BUG-002, BUG-003.
2. **Atomic order state + immutable production ledger** — BUG-004, BUG-005.
3. **Category/production validation** — BUG-006, BUG-012, BUG-014, BUG-015.
4. **Frontend production correctness** — BUG-007, BUG-013, BUG-026.
5. **Pagination, range guards, session isolation** — BUG-008, BUG-009, BUG-010.
6. **Bulk projections и journal pagination** — BUG-011, BUG-023.
7. **Admin/monitor/history correctness** — BUG-016…BUG-019, BUG-024.
8. **Auth/static resilience** — BUG-020…BUG-022.
9. **Mobile/a11y/tooling** — BUG-025, BUG-027…BUG-029.
