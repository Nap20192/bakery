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
--paper: #f4f3ee;        /* page */
--surface: #fdfcf9;      /* cards, panels */
--surface-inset: #efece4;
--ink: #191817;
--ink-soft: #4a463f;
--ink-muted: #6f6a62;    /* passes 4.5:1 on --paper for small text */
--crust: #b0532f;        /* accent; white text on it passes 4.5:1 */
--crust-dark: #8f4222;
--crust-soft: #f6e2d6;
--line: #ddd8cd;
--line-soft: #ebe7de;
--success: #4f5d26;   --success-soft: #e4ebd3;
--warning: #6e4609;   --warning-soft: #f6e7cc;
--danger: #a53e2a;    --danger-soft: #f6ddd6;
```

Rules:

- Use one terracotta accent moment per viewport.
- Never tint body text with the accent.
- Category and production-sheet colors are domain data. They must keep text,
  symbols, or dots alongside color and must not replace semantic UI tokens.

## 3. Typography

- Display/headlines and big numbers (order quantities): serif `Source Serif 4`
  (Google Fonts), fallback `Iowan Old Style`, `Georgia`. Weight 600.
- Body/UI: `Inter` (Google Fonts). Weight 400, line-height 1.55.
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
- Tables: zebra rows on `--surface-inset`; headers are mono, uppercase, tracked.
  The order matrix keeps its axes on every screen: on desktop and on a phone
  held sideways it stays a matrix with equal columns; on a portrait phone it
  becomes a day feed where the date axis turns into day headings and the shop
  axis into each panel's title. Production and summary tables keep sticky
  labels and scroll sideways inside their own frame.
- Status chips: `Отработан` (success, check icon), `Ждёт` (warning, clock
  icon), `Отменён` (danger, cross icon). In narrow matrix columns the chip
  shows its icon only; the label stays for screen readers.
- Quantity steppers: `−` value `+`, 44px buttons around a borderless number
  field on `--surface-inset`; a filled value tints the stepper (shelf order
  green, reserve amber). Used for every quantity a person enters on a phone.
- Mobile navigation: at most five tabs in the bottom bar, «Создать» as the
  raised round accent button in the middle; rarely used admin sections live
  on the profile page («Ещё»). A phone held sideways gets a left rail instead
  of the bottom bar.

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
- drop a matrix axis when converting it for a small screen: the portrait day
  feed keeps both the day and the shop on screen.
