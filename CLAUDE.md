# Bakery — расчёт теста по заявкам

## Что это

Программа для пекарни «Гагарина». Считает, сколько теста (каждого вида) нужно замесить на дату,
исходя из заявок клиентов и технологических карт (ТТК) из iiko.

## Бизнес-логика

```
Заявка (Telegram)          ТТК (iiko / excel)
  "100 булок"          ×   булка → 0.3 кг дрожж. теста / 1 шт
  "50 батонов"         ×   батон → 0.5 кг дрожж. теста / 1 шт
       ↓                        ↓
  Итого: 100×0.3 + 50×0.5 = 55 кг дрожжевого теста
```

1. Клиент отправляет заявку в Telegram → `orders` + `order_items`
2. Для каждого продукта ищем актуальную ТТК → `tech_cards` + `tech_card_items`
3. В составе ТТК находим ингредиенты с `is_dough=true`
4. `dough_per_unit = ingredient.net_qty / tech_card.yield_qty`
5. `dough_qty = order_item.quantity × dough_per_unit`
6. Группируем по типу теста → `dough_calculations`

## Источники ТТК

- **iiko (приоритет)**: через REST API `/resto/api/v2/assemblyCharts/*`
  - `getAll` — все карты за период
  - `getPrepared` — разложенная до конечных ингредиентов (оптимально для расчёта)
- **Excel (fallback)**: загрузка ТТК из скачанных файлов, если в iiko нет данных

## Структура проекта

```
cmd/main.go                  — точка входа
internal/
  domain/models.go           — доменные модели (Product, TechCard, Order, DoughCalculation …)
  iiko/
    api.go                   — формирование URL-ов iiko REST API
    client.go                — HTTP-клиент iiko (auth, products, assemblyCharts)
    consts.go                — конфигурация подключения и эндпоинты
    dto.go                   — DTO для маршалинга ответов iiko API
    client_test.go           — интеграционные тесты (ходят в реальный iiko)
migration/
  0001_init.up.sql           — схема БД (PostgreSQL)
```

## Маппинг iiko DTO → domain

| iiko DTO                | domain               | Ключевые поля                          |
|-------------------------|----------------------|----------------------------------------|
| `Product`               | `Product`            | ID→IikoID, Name, MeasureUnit→Unit      |
| `AssemblyChartDto`      | `TechCard`           | AssembledProductID, AssembledAmount→YieldQty, DateFrom/To |
| `AssemblyChartItem`     | `TechCardItem`       | ProductID→Ingredient.IikoID, AmountIn→GrossQty, AmountOut→NetQty |
| `PreparedChartDto`      | `TechCard` (resolved)| Разложенная до конечных ингредиентов    |
| `PreparedChartItem`     | `TechCardItem`       | Amount→NetQty (уже финальное)          |

## Команды

```bash
go build ./...                           # сборка
go test ./internal/iiko/ -v -run TestAuth  # тест авторизации
go test ./internal/iiko/ -v              # все интеграционные тесты
go run cmd/main.go                       # запуск
```

## Конфигурация iiko

Хост: `pekarnya-gagarina.iiko.it:443` (HTTPS)
API: iiko Resto API (`/resto/api/...`)
Авторизация: login + SHA1(password) → сессионный ключ (cookie `key=...`)

## TODO

- [ ] Загрузка ТТК из Excel (fallback)
- [ ] Telegram-бот для приёма заявок
- [ ] Сервис расчёта теста (domain logic)
- [ ] Сохранение результатов в PostgreSQL
- [ ] Синхронизация продуктов и ТТК из iiko
