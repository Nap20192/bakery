# Frontend Development Workflow

Практический процесс разработки Telegram Mini App Bakery. Это руководство
дополняет [UI kit](ui-kit.md), [feature workflow](../adding-features.md) и
[commands](../commands.md), а не заменяет их.

## Реальный стек проекта

| Область | Bakery |
|---|---|
| UI | React 19, JavaScript + `// @ts-check` для API |
| Сборка | Vite 8 |
| Стили | переход от Tailwind CSS 4 к CSS-first tokens в `src/styles.css` |
| UI primitives | `src/ui`, Radix для overlays |
| Routing | собственный History API router (`app/routes.js`, `useAppRouter`) |
| Data fetching | typed `openapi-fetch` facades в `src/api` |
| Shared state | локальные hooks; Context только при доказанной необходимости |
| Контракт | `docs/api/openapi.yaml` → `src/api/schema.d.ts` |
| Backend | Go API на `:8080`, Vite proxy `/api` |
| Browser QA | Playwright — требуется установить, см. ниже |

Не добавляйте React Router, TanStack Query, Zustand или другую UI-библиотеку
только ради унификации с типовым шаблоном. Сначала докажите, что существующий
паттерн не решает задачу.

## Шаблон задачи для Bakery

```text
Use $bakery-frontend-workflow.

Цель:
[какое действие пользователя должно стать возможным или удобнее]

Контекст:
- route: [например /orders/table]
- роли: [shop / baker / admin]
- связанные данные/API: [если известны]
- похожий экран или компонент: [если известен]
- screenshot/reference: [приложить при наличии]

Ограничения:
- сохранить существующие бизнес-правила Bakery;
- переиспользовать src/ui и tokens;
- не добавлять dependencies без объяснения;
- не менять unrelated pages или публичный API без необходимости;
- поддержать 375, 768, 1280 и 1440 px;
- смысл не должен зависеть только от цвета.

Перед кодом:
1. изучи relevant flow и перечисли файлы;
2. найди существующие аналоги;
3. объясни data/state flow;
4. задай один наводящий вопрос, только если неизвестность меняет результат;
5. предложи минимальный vertical slice.

Готово, когда:
- сценарий работает end-to-end;
- lint, typecheck, relevant tests и build прошли;
- flow проверен в реальном браузере на четырёх viewport;
- keyboard/focus, console и failed requests проверены;
- выполнен accessibility и focused diff review.

Не утверждай, что проверка прошла, если она не запускалась. В конце перечисли
точные команды, viewport и оставшиеся ограничения.
```

## Обязательный цикл

```text
Brief → исследование → дизайн-направление → план → маленький vertical slice
      → static checks → реальный браузер → accessibility → review diff
```

### 1. Brief

Перед изменением сформулируйте:

- цель пользователя и основной сценарий;
- route и роли (`shop`, `baker`, `admin`);
- данные/API и бизнес-ограничения;
- что нельзя менять;
- измеримые критерии готовности.

Наводящий вопрос задаётся, если неизвестность меняет бизнес-правило, схему БД,
API, необратимое действие или визуальное направление. Примеры:

- «Значение должно сохраняться после повторного открытия?»
- «Уже отработанные заказы исключить или открыть существующий лист?»
- «Изменение относится ко всем типам заявки или только к активному?»

Если ответ можно безопасно получить из кода или документации, вопрос не нужен.

### 2. Исследование без правок

Проверьте:

1. `AGENTS.md` и ближайшие правила.
2. Relevant route в `frontend/FRONTEND_BEHAVIOR.md`.
3. Feature-компонент, hooks и colocated model-файлы.
4. Аналоги в `src/ui` и соседних features.
5. Tokens, spacing, breakpoints и focus states.
6. API facade и OpenAPI schema.
7. Dirty worktree — чужие изменения нельзя перезаписывать.

Результат исследования: короткая архитектурная сводка, файлы, повторно
используемые компоненты, риски и план.

### 3. Дизайн-направление

Для видимого изменения используйте repo skills `interface-design` и
`typeui-fundamentals`. Зафиксируйте:

- кто пользуется экраном и в каких условиях;
- главное действие и информационную иерархию;
- поведение на 375, 768, 1280 и 1440 px;
- состояния loading, empty, error, success, disabled, hover и focus;
- как смысл дублируется без зависимости только от цвета.

Сохраняйте характер Bakery по `DESIGN.md`: serif для display/headings,
humanist sans/Inter для body и UI-хрома, cream/secondary поверхности,
тихие границы и terracotta только для главного действия. Цвет категорий
остаётся доменными данными. Избегайте glassmorphism, декоративных gradients,
emoji-icons, лишних карточек, pills и теней.

### 4. Реализация vertical slice

Реализуйте один законченный пользовательский поток. Для frontend:

- API-вызовы только через `src/api`;
- чистые преобразования — в colocated `*Model.js`/utility;
- сложный lifecycle — в feature `use*.js`;
- повторяемый UI — в `src/ui`;
- `ui` никогда не импортирует `features`;
- backend остаётся точкой авторизации и бизнес-валидации;
- новые dependencies — только с объяснением.

Контрактное изменение выполняется в порядке:

```text
backend DTO → openapi.yaml → route-sync test → api-gen → facade → UI
```

### 5. Static checks

Frontend-only:

```bash
cd frontend
npm run lint
npm run typecheck
npm run build
```

Backend/contract/SQL:

```bash
make sqlc                         # только после queries/*.sql
go test ./internal/inbound/api   # routes ↔ OpenAPI
make api-gen                     # schema.d.ts + typecheck
make build && make vet && make test && make lint
```

Нельзя писать «работает», если соответствующая команда не запускалась.

### 6. Browser loop

Запустите worker и frontend, затем проверьте полный сценарий через
`playwright-interactive`:

```bash
go run ./cmd/worker
cd frontend && npm run dev
```

Обязательные viewport:

- 375×812 — телефон;
- 768×1024 — планшет;
- 1280×800 — компактный desktop;
- 1440×900 — основной desktop.

Проверьте мышь и клавиатуру, focus visibility, длинные номера/имена,
горизонтальный overflow, sticky/fixed UI, загрузку, пустой список, ошибку API,
успех, double submit, console и failed network requests. Для визуального diff
сохраните screenshot до/после. Исправьте найденное и повторите цикл.

### 7. Accessibility

Автоматически: Playwright + axe. Вручную:

- все действия доступны с клавиатуры;
- порядок focus совпадает с визуальным;
- icon-only controls имеют имя;
- errors связаны с полями;
- цвет не является единственным носителем смысла;
- touch targets не меньше 44 px на телефоне;
- `prefers-reduced-motion` не теряет функциональность.

### 8. Diff review

Перед handoff:

```bash
git diff --check
git status -sb
git diff --stat
```

Проверьте stale state/effects, race conditions, duplicated source of truth,
пропущенные состояния, mobile overflow, fragile Tailwind classes, случайные
dependencies и unrelated changes. При staging всегда исключайте локальный DSN:

```bash
git add -A -- . ':!cmd/testingorders/main.go'
```

## Установка browser QA

### Codex skills

В приложении Codex установите два официальных curated skills:

```text
$skill-installer playwright
$skill-installer playwright-interactive
```

Или одной shell-командой:

```bash
python3 ~/.codex/skills/.system/skill-installer/scripts/install-skill-from-github.py \
  --repo openai/skills \
  --path skills/.curated/playwright skills/.curated/playwright-interactive
```

Skills станут доступны в новом turn/после обновления Codex session.

`frontend-skill` сейчас отсутствует в официальном curated-каталоге. Для
визуальной работы Bakery уже использует repo-local `interface-design` и
`typeui-fundamentals`; устанавливать ещё один дублирующий design skill не
нужно без конкретной причины.

### Project dependencies

```bash
cd frontend
npm install -D @playwright/test @axe-core/playwright
npx playwright install chromium firefox webkit
```

После установки добавьте `playwright.config.js`, папку `e2e/` и scripts
`test:e2e`, `test:e2e:ui`, `test:a11y`. Не добавляйте scripts заранее: они не
должны указывать на отсутствующие dependencies.

Полезные ссылки:

- [OpenAI skills repository](https://github.com/openai/skills)
- [Playwright skill](https://github.com/openai/skills/tree/main/skills/.curated/playwright)
- [Playwright Interactive skill](https://github.com/openai/skills/tree/main/skills/.curated/playwright-interactive)
- [Codex frontend workflows](https://developers.openai.com/codex/use-cases/frontend-designs)
- [Codex best practices](https://developers.openai.com/codex/learn/best-practices)
- [Codex skills](https://developers.openai.com/codex/skills)
- [Playwright best practices](https://playwright.dev/docs/best-practices)
- [Playwright accessibility testing](https://playwright.dev/docs/accessibility-testing)

## Handoff evidence template

```text
Changed:
- files and user-visible behavior

Validated:
- command: passed/failed
- browser: viewport + scenario
- console/network: result
- accessibility: automated + manual result

Not validated / limitations:
- exact missing tool, environment or state
```
