# Bakery — Technical Documentation

Documentation for the bakery order-management system: a Go 1.26 modular
monolith (worker + Telegram bot) over PostgreSQL 17 and RabbitMQ 3, with a
React 19 / Vite 8 / Tailwind 4 Telegram Mini App frontend. Exact pinned
versions: [architecture.md → Technology stack](architecture.md#technology-stack).

Organised loosely by the [Diátaxis](https://diataxis.fr/) framework: the
service and UI-kit pages are **reference** (look things up), the architecture
and edge-cases pages are **explanation** (understand why).

## Contents

| Document | Type | What it covers |
|---|---|---|
| [commands.md](commands.md) | How-to | Run/restart/generate commands: build, sqlc, docker, worker/bot, frontend, workflows |
| [adding-features.md](adding-features.md) | How-to | The backend↔frontend sync loop and step-by-step checklists for adding features |
| [architecture.md](architecture.md) | Explanation | System layout, layers, binaries, composition root, auth, events, database, configuration |
| [database.md](database.md) | Reference | Every sqlc query in `queries/` — purpose, parameters, callers, SQL conventions |
| [api/openapi.yaml](api/openapi.yaml) | Reference | The HTTP API contract; drives the route-sync test and the frontend's typed client |
| [services/order.md](services/order.md) | Reference | Orders, categories, dish catalog, templates, production sheets (отработка), HTTP API, events |
| [services/auth.md](services/auth.md) | Reference | Users, passwords, Telegram binding, sessions, roles |
| [services/admin.md](services/admin.md) | Reference | Admin panel backend: user provisioning and department assignment |
| [services/department.md](services/department.md) | Reference | Departments (shops and workshop) |
| [services/monitor.md](services/monitor.md) | Reference | Dough/ingredient calculation over tech-card graphs |
| [services/sync.md](services/sync.md) | Reference | iiko catalog and tech-card snapshot sync |
| [services/techcard.md](services/techcard.md) | Reference | Tech-card lookup service |
| [frontend/ui-kit.md](frontend/ui-kit.md) | Reference | Design tokens, UI components, category color system, frontend conventions |
| [frontend/improvement-plan.md](frontend/improvement-plan.md) | How-to | Prioritized readability / responsiveness / UX improvement roadmap |
| [edge-cases.md](edge-cases.md) | Explanation | Domain rules and edge cases that must not regress |

Related documents outside `docs/`:

- `AGENTS.md` — contributor guide (commands, conventions, how to extend).
- `CLAUDE.md` — condensed rules for AI-assisted work.
- `frontend/FRONTEND_BEHAVIOR.md` — full route and behavior spec of the Mini App.
- `templates/monitor_codes.txt` — dough monitoring codes per order category.
