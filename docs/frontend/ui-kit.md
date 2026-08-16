# Frontend UI Kit & Conventions

The UI is rendered by Go templates in `frontend/internal/web/templates/` and
styled by `frontend/internal/web/static/app.css`. `DESIGN.md` remains the
visual source of truth.

## Tokens

The `:root` block maps the warm editorial system into semantic variables:

- surfaces: `--paper`, `--paper-deep`, `--surface`, `--surface-inset`;
- text: `--ink`, `--ink-soft`, `--ink-muted`;
- action: `--crust`, `--crust-dark`;
- semantics: `--success`, `--warning`, `--danger` and soft companions;
- structure: `--line`, `--line-soft`, 6px controls, 8px surfaces.

Do not scatter raw colors through feature selectors. Category and production
sheet colors are the exception because they are domain identity, not UI state.

## Typography and spacing

Page and modal headings use the editorial serif stack; body, forms, and
navigation use the system sans stack; order numbers and quantities use mono.
Letter spacing is zero except the documented 2% uppercase eyebrow label.
Spacing follows an 8px baseline with 4px-compatible control adjustments.

## Shared template patterns

- `layout`: security metadata, top navigation, boosted `#app-shell`, progress
  indicator, flash region, preview slot, and toast live region.
- `flash`: safe success/error feedback returned by handlers.
- `category-badge`: color dot plus category name; never color alone.
- `order-card`: compact matrix/list item with category stripe, full number,
  item count, production/cancellation symbol, and independent `Обзор` action.
- `monitor`: reusable dough totals and expandable composition/breakdown. When a
  report has components, the total is an inline numeric input that scales the
  visible composition locally for quick «what if we need more dough» checks.
- `production-rows`: the shared production editor for new and existing sheets.
  Visible columns are `Название / Заявка / Закладка`; output is submitted as a
  hidden value inside the load cell, so hidden fields never create extra table
  columns.

## Controls

- `.button-primary` is reserved for the single consequential action in a
  local surface.
- `.button-danger` always opens the native confirmation dialog before submit.
- `.field` provides label/control association; inputs are inset relative to
  their surrounding surface.
- `.monitor-total-input` is intentionally compact: the dough total itself is the
  edit control, not a second explanatory row.
- Native `<details>` is used for disclosure and `<dialog>` for modal behavior,
  preserving keyboard and focus mechanics without a component library.

## Tables on phones

The order matrix, production journal, dough summary, and production editor do
not collapse into generic cards. Their wrappers scroll horizontally while
date/dish axes and header/footer rows stay sticky. The 375px order matrix uses
compact shop columns so three columns remain scannable.

On mobile, the bottom nav becomes fixed in the thumb zone, but it hides while a
form control has focus so Telegram/iOS keyboards do not lift it over the input.
The order editor's sticky submit bar follows the same rule.

## Category colors

Backend slugs remain `amber`, `sky`, `violet`, `emerald`, `rose`, and `stone`.
`categoryTone` validates the slug before a template adds one of the literal
CSS modifier classes. Keep the backend enum, OpenAPI enum, helper switch, and
CSS modifiers in sync.
