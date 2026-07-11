# Frontend UI Kit & Conventions

The Mini App lives in `frontend/` (React 19.2, Vite 8, Tailwind CSS 4.3 —
see the [stack table](../architecture.md#technology-stack) for exact pinned
versions). This page
documents the design system (`src/ui/`, `src/styles.css`), the category color
system, and the structural conventions of `src/`.

## Design tokens (`src/styles.css`)

Tailwind 4 is configured **CSS-first** through `@theme` — there is no
`tailwind.config.js`.

### Palette — «мука и корка» (flour & crust)

The default cool `stone` scale is **overridden** with a warm taupe ramp
derived from bread-crust tones. Components keep using `stone-*` classes and
get the warm palette for free:

| Token | Value | Typical use |
|---|---|---|
| `stone-50` | `#faf6ee` | subtle fills (`MetaCell`, count pills) |
| `stone-100` | `#f2ecdd` | hover fills, ghost hover |
| `stone-200` | `#e7dcc8` | borders of panels/cards |
| `stone-300` | `#d3c3a9` | control borders |
| `stone-400` | `#a8947a` | dashed borders, neutral dots |
| `stone-500` | `#837059` | secondary icons |
| `stone-600` | `#665441` | secondary text, labels |
| `stone-700` | `#503f30` | badge text on stone |
| `stone-800` | `#3c2f23` | button text, primary hover |
| `stone-900` | `#2d2318` | primary buttons, focus rings |
| `stone-950` | `#211911` | headings, body text |
| `flour` | `#f7f3ea` | app background (`--color-flour`) |

Functional colors are **data semantics and are not tinted**: today/tomorrow
panel accents, category colors, and red cancellation/error states keep their
Tailwind hues.

### Typography

- Font: **Golos Text Variable** (Cyrillic grotesque), self-hosted via
  `@fontsource-variable/golos-text` — no CDN.
- Base 14px / 1.5, letter-spacing `-0.01em`; headings `-0.02em`.

**Type scale — six steps, no other text sizes** (defined in `@theme`, used as
`text-caption` … `text-page`; arbitrary `text-[13px]` is a review error):

| Utility | Size/LH | Role |
|---|---|---|
| `text-caption` | 11/16 | uppercase-надзаголовки, оси таблиц — не основной текст |
| `text-note` | 12/18 | вторичные подписи: автор, даты списков |
| `text-body` | 13/20 | основной текст, кнопки |
| `text-input` | 14/20 | значения полей (на телефоне медиа-правило поднимает до 16) |
| `text-title` | 16/24 | заголовки панелей |
| `text-page` | 18/26 | заголовок экрана |

### Radii — the paper-journal rule

The interface reads like a **бумажный журнал выпечки**: calm corners, nothing
inflated. Tailwind steps are re-mapped in `@theme`:

- `rounded-md` (8px) — controls: buttons, inputs, table cells;
- `rounded-lg` = `rounded-xl` (10px) — one surface radius for cards, panels,
  dialogs;
- `rounded-full` — only data pills: category badges, sheet badges, chips,
  counters.

Don't introduce new radius values; change the mapping in `styles.css` if the
character ever changes.

### Motion & mobile

- Two keyframe utilities: `.fade-in` (0.15s) and `.pop-in` (0.16s, used by
  `Modal`). Both disabled under `prefers-reduced-motion`.
- On ≤640px every input/select/textarea is forced to **16px** font — smaller
  values make iOS Safari zoom the viewport on focus.
- Touch targets: controls are `min-h-11` on phones, `sm:min-h-9` on desktop.

## Components (`src/ui/`)

All components are small, stateless (except `CopyButton`) and accept
`className` for extension. The kit follows the **shadcn/ui pattern**:

- **Semantic tokens** (`background, foreground, card, primary, secondary,
  muted, accent, destructive, border, input, ring`) are defined in
  `styles.css` `@theme`, mapped onto the taupe palette. Kit components color
  themselves **only** through tokens (`bg-card`, `text-muted-foreground`,
  `border-input`, …) — retone the whole UI in one place.
- **`cva`** (class-variance-authority) declares component variants;
  **`cn`** (`lib/cn.js` = clsx + tailwind-merge) merges caller classes with
  conflict resolution — `className` overrides now win predictably.
- Behavioral primitives (overlays) are built on the headless **`radix-ui`**
  package: Radix provides accessibility mechanics (portal, focus trap, aria,
  keyboard), tokens provide the look. Rule: **features never import
  `radix-ui` directly** — primitives are wrapped once in `ui/`.

Data-semantic colors (category palette, today/tomorrow highlights, red
cancellation) intentionally stay outside the token system.

### `Button`

```jsx
<Button variant="primary" onClick={...}>Сохранить</Button>
```

A `cva`-based shadcn button; historical variant names map to shadcn looks:

| Variant | shadcn equivalent | Use |
|---|---|---|
| `default` | outline (`border-input bg-card`) | secondary actions |
| `primary` | default (solid `primary`) | the one main action of a view |
| `danger` | outlined destructive | destructive actions (cancel order, delete sheet) |
| `ghost` | ghost | toolbar/icon actions |

All variants share the focus-visible ring, disabled opacity, and touch-height
rules.

### `Field.jsx` → `InputField`, `SelectField`

Labelled form controls with the shared control style (white, `stone-300`
border, `stone-900` focus ring). Label is optional; all native props pass
through.

### `Panel.jsx` → `panelClass`, `PanelHeader`

- `panelClass` — the standard card surface string
  (`rounded-xl border-stone-200 bg-white shadow-sm`); apply it to any
  container div.
- `PanelHeader` — `eyebrow` (small uppercase context line), `title`, and an
  optional `count` pill rendered as «N поз.».

### `Modal`

The single overlay primitive, built on **Radix Dialog** (`radix-ui` package):
portal, focus trap, scroll lock, Escape and backdrop-click close, and aria
wiring come from the primitive; the dimmed blurred backdrop and centered
white panel (`pop-in`) are ours. Content and header are the caller's;
`maxWidthClass` overrides the default `max-w-3xl`. Used for order preview,
monitor "расшифровка", etc.

### `ConfirmDialog`

Destructive-action confirmation on **Radix AlertDialog** — the replacement
for `window.confirm`. Props: `open`, `title`, `description`, `confirmLabel`
(default «Удалить»), `onConfirm`, `onCancel`, `busy`. Focus lands on
«Отмена», Escape cancels, outside click does **not** close (an alert demands
a choice). Used for deleting dishes, categories, users, and production
sheets.

### `Icon`

Inline 24×24 stroke SVG set keyed by name: `orders, plus, users, user,
logout, eye, select, calculator, chevronLeft, chevronRight, close, star`.
Props: `size` (default 18), `strokeWidth`, `filled` (for the favorite star).
Unknown names fall back to `orders`. `aria-hidden` — always pair with a text
label or `aria-label` on the interactive parent.

### `CategoryBadge`

The colored тип-заявки pill (dot + name) shown on order cards and details.
Renders **nothing** when the order has no category (legacy orders) — callers
don't need to guard.

### `SheetBadge`

Production-sheet marker: `✓ №N` for an order baked as requested and `±K №N`
when the sheet contains K deviations for that order. The sheet-derived color
links an order to its production document, while the symbol conveys status
without relying on color alone. Worked order cards use a visibly inset muted
taupe surface with a stronger border;
orders still awaiting production remain white.
On `/orders`, `showStatus` expands the marker to the explicit labels
`✓ Отработан №N` or `± Отклонения K №N`.

### `ErrorBanner`

`<ErrorBanner error={message} />` — the shared inline error block
(`role="alert"`, red tint). Renders nothing when `error` is falsy. Every
feature surfaces `ApiError.message` through it.

### `EmptyState`

Dashed-border neutral placeholder for empty lists; `compact` reduces padding
for narrow columns (baker matrix cells).

### `MetaCell`

Small label-above-value tile used in order details meta grids (author, dates,
department).

### `CopyButton`

Button with clipboard write + 1.5s «Скопировано» feedback. `getText` may be a
value or a function; clipboard failures (insecure context) are swallowed.

## Category color system (`src/lib/categories.js`)

The backend stores a **color slug** per order category
(`orderdomain.CategoryColors`: `amber, sky, violet, emerald, rose, stone`).
`CATEGORY_COLORS` maps each slug to *literal* Tailwind class strings — they
must stay literal so the JIT compiler sees them. Each slug provides five
slots:

| Slot | Where it is applied |
|---|---|
| `stripe` | left border accent on order cards |
| `badge` | `CategoryBadge` pill |
| `dot` | the badge's color dot |
| `chipActive` | selected state of category filter chips |
| `pick` | category picker tiles in the editor |

`categoryStyle(category)` falls back to `stone` for unknown/missing colors.
**Keep this map in sync with `domain.CategoryColors`** when adding a color.

## Application structure (`src/`)

```
app/        App.jsx (view switch, auth gate), routes.js (parseRoute/pathFor)
api/        client.js (fetch wrapper) + per-resource modules (auth, orders,
            dishes, users)
lib/        auth (token storage/header), categories, format, logger,
            telegram (initData), url
features/   view code by user zone:
  auth/        Login
  shop/        ShopOrdersView, OrderList, OrderEditor (bulk text entry)
  baker/       BakerOrdersView (shop×date matrix), cards/filters/review,
               MonitorReports (+ расшифровка popup)
  orders/      shared order pages: OrdersLayout, OrdersPage, OrderDetails,
               OrderPreviewModal
  production/  ProductionJournal (list), ProductionSheet (editor)
  admin/       AdminUsers, AdminDishes (catalog + categories)
  account/     Me
ui/         the design system documented above
config/     env.js (apiBase)
```

Feature folders colocate their React components with hooks and pure projection
modules. Examples: `baker/useOrdersTable.js` owns table loading state,
`baker/ordersTableModel.js` and `baker/orderMatrix.js` contain independently
testable data transformations, while `app/useAppRouter.js` owns History API
synchronization. Components remain presentation/composition boundaries.

Testing should prefer accessible roles and labels. Stable `data-testid`
attributes are reserved for composite workflow roots (layout navigation,
matrix, order card, selection bar), using the `component-element` convention;
domain identifiers live in separate `data-*` attributes.

Rules of thumb:

- **`ui/` never imports from `features/`**; features compose `ui/`
  primitives. The only `ui → lib` dependency is `CategoryBadge → categories`.
- Routing is hand-rolled (`history.pushState` + `parseRoute`); paths:
  `/orders`, `/orders/new`, `/orders/selection`, `/orders/table`, `/calculator`, `/orders/{number}`,
  `/orders/{number}/edit`, `/orders/{number}/monitor`, `/production[/{id}]`,
  `/admin/users`, `/admin/dishes`, `/me`.
- Role-based UI: `shop` creates/edits orders; `baker` gets the matrix,
  monitoring, and production; `admin` gets everything plus admin pages. The
  backend re-enforces every rule — frontend checks are UX only.

## API client (`src/api/client.js`)

- `apiRequest(path, options)` prepends the API base, attaches the auth header
  (`tma <initData>` inside Telegram, `Bearer <token>` on web), and logs
  request/response/duration through `lib/logger`.
- Non-OK responses throw **`ApiError(message, status)`** where `message` is
  the backend's safe `{"error"}` text when present, otherwise a
  human-readable Russian fallback per status (401 «Требуется вход…», 409
  «Данные изменились…», ≥500 «Ошибка на сервере…»). Network failures throw
  `ApiError(…, 0)`.
- Non-JSON success bodies are treated as errors («Сервер вернул некорректный
  ответ») — this catches proxy/HTML error pages.
- Features render `error.message` directly in `ErrorBanner`; никогда не
  показывайте технический текст.
