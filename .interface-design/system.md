# Bakery interface system

Last updated: 2026-07-12

## Primary visual reference

Use [Anthropic field-journal style](../docs/frontend/anthropic-field-journal.md)
as the project's canonical visual system. It controls colour, typography,
spacing, elevation, radii, surfaces and shared component states.

The Figma Community file **MCP Apps for Claude (Community)** remains a
secondary reference for component anatomy and iconography only. Its previous
blue palette must not be reintroduced.

## Product direction

Bakery is an operational production journal, not a generic SaaS dashboard.
It should feel calm, dense, warm and dependable for shop staff and bakers using
a phone during active work.

Domain language: order, shop, category, batch, load, output, deviation,
production journal, dough calculation.

Signature structures:

- shop × fulfillment-date order matrix;
- production journal by date × order category;
- three production quantities: Order / Load / Output;
- category color and production-sheet identity always paired with text/symbols.

## Adaptation rule

The style reference supplies visual foundations. Bakery supplies product
semantics, Russian copy, domain data and business rules. Do not copy demo
content or replace Bakery business structures with generic cards.

When Figma and the current Bakery UI differ:

1. accessibility and domain constraints win;
2. the Anthropic field-journal style wins for visual decisions;
3. exact Figma values are mapped into Bakery semantic CSS tokens rather than
   scattered raw values;
4. existing flows remain stable unless the task explicitly changes them.

## Color roles

Map the supplied parchment palette through semantic CSS variables in
`frontend/src/styles.css`; raw values only belong in the reference layer.
Clay is exclusive to the primary consequential CTA. Category colors
(`amber`, `sky`, `violet`, `emerald`, `rose`, `stone`) remain product data and
are always accompanied by a name/dot/symbol, never used as the only status.

## Foundation choices

- Typography: Anthropic Serif for reading copy, Anthropic Sans for UI chrome
  and JetBrains Mono for technical values. Source Serif 4 and Inter are the
  bundled Cyrillic-safe fallbacks. Retain tabular mono numerals for quantities
  and IDs.
- Spacing: 4px base unit. Use the documented 4/8/12/16/24/32 scale.
- Depth: no shadows; use warm surface steps and a 1px hairline border.
- Touch targets: minimum 44px on mobile.
- Focus: visible 2px focus indicator with at least 3:1 contrast.
- Motion: 100–200ms state transitions; respect reduced motion.
- Responsive QA: 375×812, 768×1024, 1280×800 and 1440×900.

## Component rules

- Represent shared patterns in `frontend/src/ui` before creating a feature-local primitive.
- Generic badges are weighted inline text, not pills, and never the sole carrier of status.
- Inputs must remain visibly distinct from their surface at rest.
- Adjacent actions share height; button labels do not wrap.
- Tables keep their identifying axis visible on mobile through sticky headers
  or columns where the workflow requires horizontal scrolling.
- Every async surface implements loading, empty, error, disabled and success
  feedback where applicable.
- Icons clarify actions and always have visible text or an accessible name.

## Styling migration

Tailwind is being removed incrementally according to
`docs/frontend/tailwind-migration.md`. New reusable patterns use semantic CSS
variables and component-prefixed plain CSS classes. Do not create a parallel
token system during the migration.
