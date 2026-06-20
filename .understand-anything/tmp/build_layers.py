import json
layers = json.load(open('/home/vnkjd/Projects/bakery/.understand-anything/tmp/layer-assign.json'))

# Merge config into documentation -> "config-docs" to stay within 10 layers
merged_proj = layers['layer:config'] + layers['layer:documentation']

out = [
 {"id":"layer:composition-root","name":"Composition root",
  "description":"Точки входа bot и worker (cmd/) и контейнер зависимостей internal/deps вместе с internal/config — собирают и связывают все сервисы и инфраструктуру приложения.",
  "nodeIds": layers['layer:composition-root']},
 {"id":"layer:inbound-adapters","name":"Inbound-адаптеры",
  "description":"Входящие адаптеры: Telegram-бот (telebot.v3) и HTTP API (internal/inbound), принимающие внешние запросы и транслирующие их в вызовы usecase.",
  "nodeIds": layers['layer:inbound-adapters']},
 {"id":"layer:application","name":"Прикладной слой (usecase)",
  "description":"Сценарии использования и сборка сервисов (usecase и app каждого сервиса в internal/services) — оркестрируют бизнес-логику и координируют domain с инфраструктурой.",
  "nodeIds": layers['layer:application']},
 {"id":"layer:domain","name":"Доменный слой",
  "description":"Доменные модели, сущности и бизнес-правила сервисов (domain-срез internal/services), независимые от инфраструктуры и фреймворков.",
  "nodeIds": layers['layer:domain']},
 {"id":"layer:outbound-infra","name":"Outbound-инфраструктура",
  "description":"Исходящие адаптеры и реализации repository: доступ к Postgres через sqlc, клиент iiko ERP и infra-срезы сервисов (internal/outbound и internal/services/*/infra).",
  "nodeIds": layers['layer:outbound-infra']},
 {"id":"layer:shared-kernel","name":"Shared kernel",
  "description":"Общие переиспользуемые пакеты: обработка ошибок, авторизационные токены, correlation id, миграции БД, логирование и клиент RabbitMQ (internal/pkg и pkg).",
  "nodeIds": layers['layer:shared-kernel']},
 {"id":"layer:data","name":"Слой данных",
  "description":"SQL-миграции, sqlc-запросы и таблицы схемы Postgres — определяют структуру хранения и доступ к данным бэкенда.",
  "nodeIds": layers['layer:data']},
 {"id":"layer:frontend","name":"Frontend (Telegram mini-app)",
  "description":"React/Vite/Tailwind мини-приложение Telegram: api-клиент, компоненты, features, lib и конфигурация сборки фронтенда.",
  "nodeIds": layers['layer:frontend']},
 {"id":"layer:infrastructure","name":"Инфраструктура и деплой",
  "description":"Dockerfile'ы сервисов, docker-compose с описанием контейнеров (db, rabbitmq, bot, worker) и Makefile — определяют контейнеризацию и развёртывание.",
  "nodeIds": layers['layer:infrastructure']},
 {"id":"layer:config-docs","name":"Конфигурация и документация",
  "description":"Корневые конфигурационные файлы (go.mod, sqlc.yaml, golangci, окружение) и проектная документация (README, AGENTS, TODO) бэкенда.",
  "nodeIds": merged_proj},
]
# validate
total = sum(len(l['nodeIds']) for l in out)
assert total == 245, total
ids=[i for l in out for i in l['nodeIds']]
assert len(ids)==len(set(ids)), 'dup'
for l in out: assert l['nodeIds'], l['id']
import os
os.makedirs('/home/vnkjd/Projects/bakery/.understand-anything/intermediate', exist_ok=True)
json.dump(out, open('/home/vnkjd/Projects/bakery/.understand-anything/intermediate/layers.json','w'), ensure_ascii=False, indent=2)
print('layers:', len(out), 'total nodes:', total)
for l in out: print(f"  {l['id']}: {len(l['nodeIds'])}")
