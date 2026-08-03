# Claude (Anthropic) — Warm Editorial

This file is the source of truth for Bakery's current visual direction:
terracotta on cream, editorial but still operational. Figma remains a secondary
reference for component anatomy and iconography; the colors, typography,
spacing, surfaces, and component rules below win for product UI.

## 1. Visual Theme

Warm editorial. Human, considered, tactile, and deliberate. The interface
should feel like a calm production journal, not a cold SaaS dashboard or a
marketing page.

Bakery adaptation: this is still an operations app. Use the editorial language
for typography, surface tone, and restraint, while preserving dense tables,
matrices, sticky axes, and production workflow speed.

## 2. Color Palette

```css
--bg-primary: #f4f3ee;
--bg-secondary: #eeede6;
--bg-inverse: #191817;
--text-primary: #191817;
--text-secondary: #5a554e;
--text-muted: #8a847a;
--accent: #c96442;
--accent-hover: #b55738;
--accent-soft: #e89268;
--border: #d8d3c8;
--success: #6b7a3d;
--warning: #c98a42;
--danger: #a53e2a;
```

Rules:

- Use one terracotta accent moment per viewport.
- Never tint body text with the accent.
- Category and production-sheet colors are domain data. They must keep text,
  symbols, or dots alongside color and must not replace semantic UI tokens.

## 3. Typography

- Display/headlines: serif, `Tiempos Headline`, `Iowan Old Style`, fallback
  `Georgia`. Weight 500.
- Body: humanist sans, `Styrene A`, fallback bundled `Inter`. Weight 400,
  line-height 1.6.
- UI/labels: same sans, weight 500, letter-spacing +2% only for uppercase
  eyebrow labels.
- Mono: `GT America Mono`, fallback bundled `JetBrains Mono`, then `SF Mono`.
- Type scale: 13 / 15 / 17 / 21 / 26 / 32 / 40 / 50 / 62.
- Use `text-wrap: balance` for headings and `text-wrap: pretty` for readable
  copy.

## 4. Components

- Buttons: primary uses terracotta fill, cream text, 6px radius, 10px/18px
  padding, 500 weight, darker terracotta on hover, no lift.
- Secondary buttons: 1px `--border`, ink text, transparent fill, secondary
  surface on hover.
- Ghost actions: ink text, no container. Inline links underline; button labels
  do not underline.
- Cards/panels: `--bg-secondary`, no default border, no shadow, 8px radius,
  24px padding. A subtle border may appear on hover or for structure.
- Inputs: 1px `--border`, 6px radius, 10px/14px padding. Focus is a 2px
  terracotta outline with 2px offset. No glow.
- Navigation: horizontal where space allows, underline-on-hover, active accent
  underline, no pill backgrounds.
- Tables: zebra rows on `--bg-secondary`; headers are mono, uppercase, tracked.
  Bakery production/order matrices keep their axes visible on mobile via
  sticky labels or horizontal scroll instead of becoming generic cards.

## 5. Layout

- App shell max width: 1180px.
- Long-form max width: 680px.
- Grid: 12 columns, 24px gutters where useful.
- Vertical rhythm: 8px baseline; section breaks around 96px.
- Mobile: single-column by default; keep 44px touch targets.
- Large serif headings scale from 62px desktop to 36px mobile.

## 6. Depth

Flat by default. Depth comes from:

- surface color shifts;
- 1px borders;
- type weight contrast.

Exception: modals may use `0 8px 24px rgba(25, 24, 23, 0.08)`.

## 7. Do and Do Not

Do:

- let whitespace and type hierarchy do the work;
- use serif for page/modal headings and sans for body/UI;
- write concrete Russian microcopy in full sentences when a state needs text;
- keep operational screens dense enough for repeated shop/workshop use.

Do not:

- use purple gradients, glassmorphism, neon glows, or decorative blobs;
- apply Inter as a visual default to every role without serif headings;
- animate hover states with scale or lift;
- mix more than two non-mono font families per page;
- use emoji in UI chrome;
- convert load-bearing tables into card stacks when the matrix axis is the
  core workflow.
