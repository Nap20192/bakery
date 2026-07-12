# Tailwind removal plan

Tailwind is removed incrementally. Every phase must preserve the rendered UI,
pass frontend checks, and be verified in a real browser before the next phase.
Do not remove `@tailwindcss/vite`, `tailwindcss`, `@theme`, or `tailwind-merge`
while feature files still contain utilities.

## Current state

- [x] Phase 1 — plain-CSS design tokens and shared primitives.
  `Button`, `Field`, `Panel`, `EmptyState`, `ErrorBanner`, and `MetaCell` now
  use `src/ui/ui.css`. `class-variance-authority` was removed because it no
  longer had callers.
- [ ] Phase 2 — remaining UI primitives and overlays: badges, modal,
  confirmation dialog, copy button, icons and sheet marker.
- [ ] Phase 3 — application shell: login, header, desktop navigation, mobile
  navigation, account screen and shared route states.
- [ ] Phase 4 — shop flow: order list, details and editor.
- [ ] Phase 5 — baker flow: matrix, selection, monitor and calculator.
- [ ] Phase 6 — production journal and sheet editor.
- [ ] Phase 7 — admin users and catalog.
- [ ] Phase 8 — complex orders table and category/date state styling.
- [ ] Phase 9 — remove compatibility layer and dependencies.

## Rules for each phase

1. Use component-prefixed classes (`ui-*`, `orders-*`, `production-*`) and
   colocate feature CSS next to the owning feature.
2. Read values from semantic CSS variables; do not introduce arbitrary colors,
   spacing or radii.
3. Preserve hover, active, focus-visible, disabled, loading, empty and error
   states. Mobile controls remain at least 44px high.
4. Do not combine behavioral refactoring with styling migration unless it is
   required to preserve the same component API.
5. Run `npm run lint`, `npm run typecheck`, `npm run build`, `git diff --check`
   and browser checks at 375×812, 768×1024, 1280×800 and 1440×900.

## Final removal gate

Before phase 9, these searches must return no application matches:

```bash
rg 'className=.*(?:flex|grid|p-|m-|text-|bg-|border-|rounded-|sm:|md:|lg:)' frontend/src
rg '@theme|@import "tailwindcss"' frontend/src
rg 'tailwind-merge|@tailwindcss/vite|tailwindcss' frontend/src frontend/package.json frontend/vite.config.js
```

Then remove the Tailwind Vite plugin, Tailwind packages, `tailwind-merge`, the
temporary `@theme` compatibility block, and simplify `cn` to `clsx` only.
