# Bakery — реестр найденных дефектов

Аудит выполнен 29 июля 2026 года на ветке `development`, commit `624d5ee`.
Все пункты ниже открыты и подтверждены статически, браузером или на отдельной
тестовой БД. Исходный код и рабочие данные во время аудита не изменялись.

Приоритеты:

- **P0** — риск порчи данных, нарушения безопасности или полной блокировки
  основного бизнес-процесса;
- **P1** — неверный результат, потеря доступности или сломанный основной UI-flow;
- **P2** — производительность, устойчивость и ошибки вторичных сценариев;
- **P3** — доступность, небольшие UX-дефекты и пробелы tooling.

Закрывать пункт можно только вместе с регрессионным тестом. План и порядок
работ находятся в [`todo.md`](todo.md). Старый [`TODO.md`](TODO.md) остаётся
отдельным архитектурным roadmap.

## P0 — целостность данных и безопасность

### BUG-001 — один заказ может попасть в две отработки

- **Факт:** из 10 параллельных `POST /production` два вернули `201`; один заказ
  оказался сразу в двух `production_sheet_orders`.
- **Причина:** проверка и вставка разделены, а в БД уникальна только пара
  `(sheet_id, order_id)`.
- **Исправлять:** [`migrations/00026_production_sheet_orders.sql`](migrations/00026_production_sheet_orders.sql),
  [`queries/production.sql`](queries/production.sql),
  [`internal/services/order/infra/repo/production_repo.go`](internal/services/order/infra/repo/production_repo.go).
- **Готово, когда:** в БД действует `UNIQUE(order_id)`, конфликт не скрывается
  через `ON CONFLICT DO NOTHING`, а параллельный integration-тест получает
  ровно один успех и остальные `409 order.production_exists`.

### BUG-002 — создание пользователя перезаписывает существующий аккаунт

- **Факт:** повторный `POST /users` с тем же логином сохранил прежний id, но
  заменил пароль, роль, отдел и Telegram username; старый пароль перестал работать.
- **Причина:** create-query реализован как upsert по `username`.
- **Исправлять:** [`queries/auth.sql`](queries/auth.sql),
  [`internal/services/auth/infra/repo/auth_repo.go`](internal/services/auth/infra/repo/auth_repo.go),
  [`internal/services/admin/infra/http/handler.go`](internal/services/admin/infra/http/handler.go);
  idempotent `EnsureAdminUser` должен использовать отдельный явный путь.
- **Готово, когда:** admin-create существующего логина возвращает `409`, не
  меняет ни одного поля и не мешает безопасному startup-ensure администратора.

### BUG-003 — одинаковые буквы категорий блокируют создание заказов

- **Факт:** вторая категория с буквой `Х` создаётся, но её первый заказ
  постоянно падает с `500`: номер конфликтует, транзакция откатывает счётчик,
  повтор снова пытается создать `.001`.
- **Причина:** уникален `code`, но не нормализованная `letter`; счётчик ведётся
  по category id, а в видимый номер входит только буква.
- **Исправлять:** [`migrations/00019_order_categories.sql`](migrations/00019_order_categories.sql),
  новая миграция, [`internal/services/order/usecase/order/order.go`](internal/services/order/usecase/order/order.go),
  [`internal/services/order/domain/service.go`](internal/services/order/domain/service.go).
- **Готово, когда:** дубликат `upper(trim(letter))` отклоняется понятным `409`
  на уровне БД и usecase; миграция заранее проверяет существующие данные.

### BUG-004 — отменённый заказ можно изменить в гонке

- **Факт:** параллельные cancel и PUT оба вернули `200`; outbox записал
  `order.cancelled`, затем `order.updated`, а отменённый заказ получил новые
  количество и комментарий.
- **Причина:** состояние проверяется до транзакции, `UPDATE` не содержит
  `cancelled_at IS NULL` и не блокирует строку.
- **Исправлять:** [`internal/services/order/usecase/order/order.go`](internal/services/order/usecase/order/order.go),
  [`queries/orders.sql`](queries/orders.sql),
  [`internal/services/order/infra/repo/order_repo.go`](internal/services/order/infra/repo/order_repo.go),
  аналогичную проверку внутри транзакции отработки в
  [`production_repo.go`](internal/services/order/infra/repo/production_repo.go).
- **Готово, когда:** update/production атомарно конфликтуют с cancel, а
  параллельные integration-тесты не создают событий после отмены.

### BUG-005 — редактирование заказа ломает сохранённую отработку

- **Факт:** после переименования позиции сохранённый лист с двумя позициями
  стал отдавать одну; старое значение закладки осталось сиротой в БД.
- **Причина:** позиции заказа заменяются целиком, а журнал связан с ними
  нормализованным `product_name` и читает их через inner join.
- **Исправлять:** [`internal/services/order/infra/repo/order_repo.go`](internal/services/order/infra/repo/order_repo.go),
  [`queries/production.sql`](queries/production.sql),
  [`migrations/00027_production_loaded_quantity.sql`](migrations/00027_production_loaded_quantity.sql).
- **Готово, когда:** отработка является неизменяемым снимком и читается
  независимо от будущих правок заказа. Допустимый краткосрочный guard —
  запрет изменения состава уже отработанного заказа, но постоянное решение
  должно сохранять идентичность/снимок позиции журнала.

### BUG-006 — категория заказа не сверяется с категорией блюда

- **Факт:** API принял заказ категории «Булочки» с блюдом «Хлеб Бородино»;
  мониторинг выбрал коды теста булочек и посчитал неверный продукт.
- **Причина:** категория и позиции разрешаются независимо.
- **Исправлять:** [`internal/services/order/usecase/order/order.go`](internal/services/order/usecase/order/order.go),
  DTO каталога и fake repository в тестах.
- **Готово, когда:** категоризированное блюдо допустимо только в своей
  категории; блюда без категории остаются доступными везде; правило работает
  для create, update и draft.

## P1 — бизнес-корректность и доступность

### BUG-007 — редактор отработки незаметно сохраняет старый выход

- **Факт:** для заявки `10`, сохранённых закладки/выхода `8/8` после открытия
  листа и изменения закладки на `7` скрытый выход остаётся `8`.
- **Причина:** строка считается связанной только при `loaded == ordered` и
  `produced == ordered`, хотя обычная связь определяется `produced == loaded`.
- **Исправлять:** [`frontend/internal/web/production.go`](frontend/internal/web/production.go),
  [`frontend/internal/web/static/app.js`](frontend/internal/web/static/app.js),
  тесты `buildProductionRows` и формы.
- **Готово, когда:** обычный сохранённый выход продолжает следовать за
  закладкой, а legacy-выход, который действительно отличался от закладки,
  остаётся зафиксированным согласно
  [`frontend/FRONTEND_BEHAVIOR.md`](frontend/FRONTEND_BEHAVIOR.md).

### BUG-008 — матрица и таблица молча обрезают заказы после первых 100

- **Факт:** обе страницы запрашивают `limit=100` и игнорируют `TotalPages`;
  таблица дополнительно фильтрует категорию уже после получения первой
  нефильтрованной страницы.
- **Исправлять:** [`frontend/internal/web/orders.go`](frontend/internal/web/orders.go),
  [`frontend/internal/web/orders_table.go`](frontend/internal/web/orders_table.go),
  application/backend pagination.
- **Готово, когда:** 101-й и последующие заказы участвуют в карточках, итогах,
  маркерах отработки и ссылках выбора; category filter передаётся worker.

### BUG-009 — произвольный диапазон дат может исчерпать память

- **Факт:** authenticated URL с `0001-01-01..9999-12-31` заставляет
  `buildOrderMatrix` создать около 3,65 млн строк на каждый отдел.
- **Исправлять:** [`frontend/internal/web/orders.go`](frontend/internal/web/orders.go).
- **Готово, когда:** диапазон валидируется и ограничивается продуктовым окном
  матрицы; неверный или слишком большой диапазон даёт `400` либо безопасный
  пятидневный default до начала построения.

### BUG-010 — выбранная партия протекает между аккаунтами

- **Факт:** baker A выбрал заказ, вышел, baker B вошёл в той же вкладке и
  получил selection mode, подсветку и ссылку на партию A.
- **Причина:** origin-global ключи и memory cache не привязаны к viewer и не
  очищаются logout.
- **Исправлять:** [`frontend/internal/web/static/app.js`](frontend/internal/web/static/app.js),
  [`frontend/internal/web/handlers_auth.go`](frontend/internal/web/handlers_auth.go),
  при необходимости viewer data в [`layout.html`](frontend/internal/web/templates/layout.html).
- **Готово, когда:** selection namespace содержит стабильный viewer id или
  очищается при любой смене сессии; browser-тест переключает двух пользователей.

### BUG-011 — страницы партии делают до сотен последовательных HTTP-вызовов

- **Факт:** `/orders/selection` делает `M+1` вызовов, detail листа — `K+2`,
  журнал — `E+3`, где `E` растёт без пагинации. Ошибка lookup в журнале
  проглатывается, и лист исчезает из успешного ответа.
- **Исправлять:** [`frontend/internal/web/orders.go`](frontend/internal/web/orders.go),
  [`frontend/internal/web/production.go`](frontend/internal/web/production.go),
  [`frontend/internal/backend/resources.go`](frontend/internal/backend/resources.go),
  [`internal/services/order/infra/http/handler_production.go`](internal/services/order/infra/http/handler_production.go),
  [`queries/production.sql`](queries/production.sql),
  [`docs/api/openapi.yaml`](docs/api/openapi.yaml).
- **Готово, когда:** selection заранее ограничен максимумом, данные партии
  загружаются bulk-запросом, журнал пагинируется и уже содержит категорию,
  а backend error отображается вместо тихого пропуска листа.

### BUG-012 — отработка допускает заказ без единой позиции

- **Факт:** `items: []` создаёт лист и membership заказа, но не сохраняет ни
  одной закладки.
- **Исправлять:** [`internal/services/order/usecase/order/order.go`](internal/services/order/usecase/order/order.go),
  [`internal/services/order/infra/repo/production_repo.go`](internal/services/order/infra/repo/production_repo.go).
- **Готово, когда:** каждый item выбранного заказа встречается ровно один раз;
  пропуск, неизвестная или дублированная позиция дают `400`.

### BUG-013 — пропорциональное округление может создать отрицательную долю

- **Факт:** распределение общего количества `1` по 12 равным заказам даёт
  одиннадцать долей `0.1` и последнюю `-0.1`; worker затем отвергает форму.
- **Исправлять:** [`frontend/internal/web/production.go`](frontend/internal/web/production.go).
- **Готово, когда:** распределение в целых десятых использует безопасный
  остаток, каждая доля неотрицательна, а сумма точно равна total.

### BUG-014 — смешанную категорию можно сохранить в одной отработке

- **Факт:** production вернул `201` для bread+buns, после чего тот же batch
  monitor вернул `400`.
- **Исправлять:** [`internal/services/order/usecase/order/order.go`](internal/services/order/usecase/order/order.go);
  monitor guard в
  [`internal/services/monitor/infra/http/handler.go`](internal/services/monitor/infra/http/handler.go)
  остаётся дополнительной защитой.
- **Готово, когда:** production API атомарно отклоняет смешанные категории.

### BUG-015 — пекарь может сменить источник своего заказа на магазин

- **Факт:** create принудительно использует «Цех Пекари», но PUT того же
  пекаря успешно меняет источник на магазин, сохраняя номер `Ц.*`.
- **Исправлять:** [`internal/services/order/infra/http/handler_write.go`](internal/services/order/infra/http/handler_write.go).
- **Готово, когда:** update применяет то же правило, что create, либо неизменно
  сохраняет workshop source для baker-created order.

### BUG-016 — PATCH пользователя частично коммитит неуспешный запрос

- **Факт:** `{username: new, role: invalid}` вернул `400`, но новый username
  уже сохранился.
- **Причина:** поля обновляются отдельными вызовами без общей транзакции.
- **Исправлять:** [`internal/services/admin/infra/http/handler.go`](internal/services/admin/infra/http/handler.go),
  admin usecase/ports и auth repository.
- **Готово, когда:** весь patch сначала валидируется и сохраняется одной
  транзакцией; при любой ошибке не меняется ни одно поле.

### BUG-017 — изменение code категории возвращает ложный успех

- **Факт:** PUT с новым code вернул `200`, но response и БД сохранили старый.
- **Исправлять:** [`queries/categories.sql`](queries/categories.sql),
  [`internal/services/order/infra/repo/order_repo.go`](internal/services/order/infra/repo/order_repo.go).
- **Готово, когда:** code либо действительно обновляется с проверкой уникальности,
  либо объявлен immutable и удалён из update DTO/form.

### BUG-018 — цикл техкарт превращается в ошибку до доменного расчёта

- **Факт:** repository возвращает cycle error, хотя domain и edge-case
  договорённость требуют нулевой вклад повторной ветки.
- **Исправлять:** [`internal/services/monitor/infra/repo/monitor_repo.go`](internal/services/monitor/infra/repo/monitor_repo.go),
  тесты [`internal/services/monitor/domain`](internal/services/monitor/domain).
- **Готово, когда:** загрузчик безопасно обрывает повторную ветку, сохраняет
  depth limit 12 и не меняет расчёты ациклического графа.

### BUG-019 — отсутствие iiko row создаёт ложную историю removed+added

- **Факт:** неизменённое блюдо с пустым прочитанным code записывается в историю
  как удалённое и добавленное.
- **Причина:** diff использует только product code, который nullable на записи
  и пуст при чтении без iiko product.
- **Исправлять:** [`internal/services/order/infra/repo/order_repo.go`](internal/services/order/infra/repo/order_repo.go),
  [`internal/services/order/usecase/order/order.go`](internal/services/order/usecase/order/order.go).
- **Готово, когда:** у позиции есть стабильный ключ или безопасный fallback по
  нормализованному имени; неизменённый заказ не создаёт историю.

## P2 — безопасность, производительность и устойчивость

### BUG-020 — login/session exchange освобождён от CSRF и не throttled

- **Факт:** hostile-origin form может заменить сессию пользователя аккаунтом
  атакующего; каждый публичный login запускает PBKDF2 без application-side limit.
- **Исправлять:** [`frontend/internal/web/server.go`](frontend/internal/web/server.go),
  [`frontend/internal/web/templates/pages/login.html`](frontend/internal/web/templates/pages/login.html),
  [`frontend/internal/web/handlers_auth.go`](frontend/internal/web/handlers_auth.go),
  login middleware/deploy edge configuration.
- **Готово, когда:** login и Telegram exchange требуют CSRF/Origin validation,
  token ротируется после входа, а burst неверных попыток получает `429` или
  эквивалентный подтверждённый edge-limit.

### BUG-021 — Telegram script блокирует HTMX и app.js

- **Факт:** при задержке Telegram на 1.608 s локальные файлы уже скачаны, но
  `window.htmx` остаётся undefined и `DOMContentLoaded` ждёт внешний script.
- **Исправлять:** [`frontend/internal/web/templates/layout.html`](frontend/internal/web/templates/layout.html),
  [`frontend/internal/web/static/app.js`](frontend/internal/web/static/app.js).
- **Готово, когда:** локальный UI стартует независимо от Telegram CDN; Telegram
  bridge подключается неблокирующе и сохраняет Mini App login.

### BUG-022 — static assets отдаются без cache policy и compression

- **Факт:** CSS/HTMX/app.js — около 109 KB raw; ответы не имеют
  `Cache-Control`, `ETag`, `Last-Modified` и `Content-Encoding`.
- **Исправлять:** [`frontend/internal/web/server.go`](frontend/internal/web/server.go),
  asset URLs в [`layout.html`](frontend/internal/web/templates/layout.html),
  при необходимости Railway/proxy configuration.
- **Готово, когда:** versioned assets безопасно кешируются, повторный reload
  получает cache hit/304, а production response с `Accept-Encoding` сжимается.

### BUG-023 — ошибка backend вызывает повторный `/me` и удваивает timeout

- **Факт:** `requireViewer` получает ошибку `/me`, затем `renderError` снова
  вызывает `viewer`; два 20-секундных ожидания превышают 30-секундный write timeout.
- **Исправлять:** [`frontend/internal/web/server.go`](frontend/internal/web/server.go).
- **Готово, когда:** error rendering не выполняет повторную backend-аутентификацию
  и отправляет полезный `502` в пределах одного timeout.

### BUG-024 — reorder допускает повторную отправку

- **Факт:** boosted form не синхронизирует/не блокирует повторный submit;
  второй запрос читает уже изменённый порядок и может сдвинуть блюдо дважды.
- **Исправлять:** [`frontend/internal/web/templates/components/form.html`](frontend/internal/web/templates/components/form.html),
  [`frontend/internal/web/admin.go`](frontend/internal/web/admin.go).
- **Готово, когда:** UI отправляет одну команду, а повтор той же операции
  идемпотентен относительно желаемого порядка.

## P2 — пользовательские сценарии

### BUG-025 — форма нового заказа шире мобильного viewport

- **Факт:** на 375×812 документ имеет `clientWidth=375`, `scrollWidth=483`;
  sticky footer — 471 px, submit частично находится за экраном.
- **Исправлять:** [`frontend/internal/web/templates/pages/order_form.html`](frontend/internal/web/templates/pages/order_form.html),
  [`frontend/internal/web/static/app.css`](frontend/internal/web/static/app.css).
- **Готово, когда:** `/orders/new` не имеет document-level overflow на 320/375 px,
  а все действия остаются видимыми и не уже 44 px.

### BUG-026 — detail заказа не показывает обязательный факт выпечки

- **Факт:** таблица содержит только «Заказ/Резерв», игнорируя
  `ProducedQuantity` и `ProducedReason`; shop также видит ссылку на production,
  которую затем получает `403`.
- **Исправлять:** [`frontend/internal/web/templates/pages/order_detail.html`](frontend/internal/web/templates/pages/order_detail.html),
  view model/format helpers и CSS состояний.
- **Готово, когда:** колонка «Испечено» показывает факт, зелёный/красный delta
  и reason; ссылка на лист интерактивна только для baker/admin.

### BUG-027 — frontend теряет или показывает устаревшее состояние

- **Подслучаи:**
  - calculator не очищает старый result после смены категории;
  - `category_id=0` («Без типа») сбрасывается на первую обычную категорию;
  - pagination заказов теряет department/date filters;
  - table category tab теряет `start`;
  - ошибки загрузки и удаления draft проглатываются.
- **Исправлять:** [`frontend/internal/web/static/app.js`](frontend/internal/web/static/app.js),
  [`frontend/internal/web/orders_table.go`](frontend/internal/web/orders_table.go),
  [`frontend/internal/web/templates/pages/orders.html`](frontend/internal/web/templates/pages/orders.html),
  [`frontend/internal/web/templates/pages/orders_table.html`](frontend/internal/web/templates/pages/orders_table.html),
  [`frontend/internal/web/orders.go`](frontend/internal/web/orders.go).
- **Готово, когда:** каждое состояние сохраняется в ссылках/HTMX navigation,
  а backend failure показывается пользователю и не выглядит как пустой draft.

## P3 — доступность и tooling

### BUG-028 — набор accessibility-дефектов

- **Подслучаи:**
  - white on `#cc785c` — `3.28:1`, focus — `2.89:1`, input boundary — около `1.01:1`;
  - preview `30×30`, comment control высотой `32`, view tabs около `37` px;
  - selection card становится `role=button`, сохраняя focusable link/button внутри;
  - нет skip link;
  - confirm dialog не имеет `aria-label`/`aria-labelledby`;
  - focus hidden radio «Обзор» рисуется на tab «Отработка»;
  - login имеет только `h2` и пустую desktop-колонку.
- **Исправлять:** [`frontend/internal/web/static/app.css`](frontend/internal/web/static/app.css),
  [`frontend/internal/web/templates/layout.html`](frontend/internal/web/templates/layout.html),
  [`frontend/internal/web/templates/components/order_card.html`](frontend/internal/web/templates/components/order_card.html),
  [`frontend/internal/web/templates/pages/login.html`](frontend/internal/web/templates/pages/login.html),
  [`frontend/internal/web/static/app.js`](frontend/internal/web/static/app.js).
- **Готово, когда:** WCAG AA contrast/focus проходит, touch targets не меньше
  44×44, DOM не содержит вложенных interactive controls, полная клавиатурная
  навигация и Axe не находят blocker/critical issues.

### BUG-029 — lint не проверяет frontend

- **Факт:** `.golangci.yml` исключает `frontend/`; общий lint отдельно падает
  на G101 для тестового dummy DSN.
- **Исправлять:** [`.golangci.yml`](.golangci.yml),
  [`internal/outbound/db/postgres_test.go`](internal/outbound/db/postgres_test.go).
- **Готово, когда:** frontend входит в lint scope, dummy credential помечен
  узким обоснованным suppression или заменён безопасным fixture, `make lint`
  сообщает 0 issues.

## Уже проверено

- `go test ./...`
- `go vet ./...`
- focused `go test -race` для frontend/order/monitor
- production build frontend
- OpenAPI route/spec sync
- `/orders` в 375×812, 768×1024, 1280×800 и 1440×900

Полный keyboard traversal и автоматический Axe scan в исходном аудите не
завершались; они обязательны перед закрытием BUG-028.
