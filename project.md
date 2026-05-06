# Bakery Project Overview / Описание проекта Bakery

## English

### What this project is
Bakery is a Go-based automation system for a bakery workflow.  
It combines:
1. Telegram bots for order intake and administration
2. iiko API synchronization (products + assembly/prepared charts)
3. SQLite storage with sqlc-generated repository layer
4. Role-based access control (RBAC) for bot actions

### Main runtime components
- `cmd/orderbot` — customer-side Telegram bot (creating orders).
- `cmd/adminbot` — staff/admin Telegram bot (viewing orders, operational commands).
- `internal/app/sync.go` — periodic iiko sync job running in process with bots.
- `internal/repo/sqlc` — generated SQL data-access layer.
- `migrations/00001_init.sql` — database schema.

### Domain and business logic
- **Auth** (`internal/app/auth.go`, `internal/domain/auth.go`)
  - Telegram/password user records.
  - Roles: `admin`, `baker`, `client`.
  - Role normalization/validation.
- **Orders** (`internal/app/order.go`)
  - Create order with generated number (`DDMMYYYY_ORDER_XXXX`).
  - Store order items.
  - Fetch single order and list recent orders.
  - Resolve dish by product name from synced iiko catalog.
- **Ingredient monitoring/reporting** (`internal/app/monitor.go`, `internal/domain/monitor.go`)
  - For a selected ingredient, calculates total required quantity across an order.
  - Builds per-dish breakdown and proportion in total consumption.
- **Sync** (`internal/app/sync.go`)
  - Authenticates in iiko.
  - Pulls products and charts.
  - Saves snapshots into local DB in one transaction.

### Bots and permissions
- **OrderBot** (`internal/bot`)
  - Parses multiline bulk order messages.
  - Shows preview and confirmation flow.
  - Creates order after confirmation.
- **AdminBot**
  - Shows latest orders.
  - Looks up specific orders by number.
  - Includes operational command entry points (`/accept`, `/delete`, `/close`, `/reports`, `/groups_add`, `/users_add`).

RBAC middleware (`internal/bot/middleware.go`) maps permissions:
- `client`: create orders
- `baker`: view orders/reports, accept/delete/close orders
- `admin`: all above + manage groups/users

### Storage model (SQLite)
Key tables:
- `auth_users`, `orders`, `order_items`, `order_counters`
- `iiko_products`
- `iiko_assembly_charts`, `iiko_assembly_chart_items`
- `iiko_prepared_charts`, `iiko_prepared_chart_items`
- `iiko_sync_runs`

### Configuration
Environment-based (`internal/config/config.go`, `.env.example`):
- Telegram tokens
- iiko host/credentials
- sync interval and date window
- DB path and HTTP port

### Tooling
- `sqlc` for typed SQL access (`queries/*.sql` -> `internal/repo/sqlc/*.go`)
- `goose` for migrations (`Makefile` targets: `migrate-up`, `migrate-down`, `migrate-status`, `migrate-reset`)

### Current status notes
- Core architecture and RBAC are present.
- Some operational/admin commands are currently scaffolds and need full business implementation.
- There are known unrelated build inconsistencies in some `cmd/*` entrypoints in the current branch.

---

## Русский

### Что это за проект
Bakery — это система автоматизации для пекарни на Go.  
В проекте объединены:
1. Telegram-боты для приёма и администрирования заказов
2. Синхронизация с iiko API (номенклатура + техкарты)
3. SQLite-база с data-access слоем через sqlc
4. Ролевая модель доступа (RBAC) для действий в ботах

### Основные компоненты запуска
- `cmd/orderbot` — клиентский бот (создание заказов).
- `cmd/adminbot` — бот для персонала/админа (просмотр и операционные команды).
- `internal/app/sync.go` — фоновая периодическая синхронизация iiko.
- `internal/repo/sqlc` — сгенерированный слой запросов к БД.
- `migrations/00001_init.sql` — схема БД.

### Домен и бизнес-логика
- **Авторизация** (`internal/app/auth.go`, `internal/domain/auth.go`)
  - Пользователи Telegram/пароль.
  - Роли: `admin`, `baker`, `client`.
  - Нормализация и проверка ролей.
- **Заказы** (`internal/app/order.go`)
  - Создание заказа с номером формата `DDMMYYYY_ORDER_XXXX`.
  - Сохранение позиций заказа.
  - Получение заказа по номеру и списка последних заказов.
  - Поиск блюда по названию в синхронизированной номенклатуре iiko.
- **Мониторинг ингредиентов** (`internal/app/monitor.go`, `internal/domain/monitor.go`)
  - Для выбранного ингредиента считает общий требуемый объём по заказу.
  - Строит детализацию по блюдам и долю каждого блюда в общем расходе.
- **Синхронизация** (`internal/app/sync.go`)
  - Авторизация в iiko.
  - Загрузка номенклатуры и техкарт.
  - Сохранение снапшота в БД в одной транзакции.

### Боты и права доступа
- **OrderBot** (`internal/bot`)
  - Разбирает многострочные batch-заявки.
  - Делает предпросмотр и подтверждение.
  - Создаёт заказ после подтверждения.
- **AdminBot**
  - Показывает последние заказы.
  - Ищет конкретные заказы по номеру.
  - Содержит точки входа для операций (`/accept`, `/delete`, `/close`, `/reports`, `/groups_add`, `/users_add`).

RBAC middleware (`internal/bot/middleware.go`) задаёт права:
- `client`: создавать заказы
- `baker`: смотреть заказы/отчёты, принимать/удалять/закрывать заказы
- `admin`: всё перечисленное + управление группами/пользователями

### Модель хранения (SQLite)
Ключевые таблицы:
- `auth_users`, `orders`, `order_items`, `order_counters`
- `iiko_products`
- `iiko_assembly_charts`, `iiko_assembly_chart_items`
- `iiko_prepared_charts`, `iiko_prepared_chart_items`
- `iiko_sync_runs`

### Конфигурация
Через переменные окружения (`internal/config/config.go`, `.env.example`):
- токены Telegram
- host/учётные данные iiko
- интервал и окно дат синхронизации
- путь к БД и HTTP-порт

### Инструменты
- `sqlc` для типобезопасного доступа к SQL (`queries/*.sql` -> `internal/repo/sqlc/*.go`)
- `goose` для миграций (`Makefile`: `migrate-up`, `migrate-down`, `migrate-status`, `migrate-reset`)

### Текущий статус
- Базовая архитектура и RBAC уже внедрены.
- Часть админских операционных команд пока каркасная и требует полной бизнес-реализации.
- В текущей ветке есть известные несвязанные несоответствия сборки в некоторых `cmd/*`.
