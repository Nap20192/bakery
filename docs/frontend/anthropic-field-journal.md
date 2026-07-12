# Anthropic field-journal style

Current visual source of truth for Bakery, supplied by the product owner on
2026-07-12. It supersedes the previously adopted blue Figma palette for all
shared surfaces and controls. The Figma Community file remains a reference for
component anatomy and iconography only.

## Intent

Bakery is an operational production journal. It should feel like a quiet field
notebook: warm parchment surfaces, editorial reading copy, compact sans-serif
controls and one clay action colour. It is not a generic blue SaaS dashboard.

## Foundation tokens

| Role | Token | Value |
|---|---|---|
| Main ink | `--color-slate-dark` | `#141413` |
| Canvas | `--color-ivory-medium` | `#f0eee6` |
| Card | `--color-ivory-light` | `#faf9f5` |
| Quiet text | `--color-cloud-medium` | `#b0aea5` |
| Control border | `--color-cloud-dark` | `#87867f` |
| Hairline border | `--color-stone` | `#cccbc8` |
| Grouped surface | `--color-oat-warm` | `#e3dacc` |
| Featured surface | `--color-manilla` | `#f5e3c7` |
| Consequential CTA | `--color-clay` | `#d97757` |
| CTA hover | `--color-clay-deep` | `#c6613f` |

`Anthropic Serif` is represented by the locally bundled, Cyrillic-safe Source
Serif 4 fallback; `Anthropic Sans` by Inter; `Anthropic Mono` by JetBrains
Mono. Do not use the supplied Anthropic Sans archive until a licensed
Cyrillic-capable face is supplied.

## Component rules

- Canvas is ivory, cards are light ivory, and grouped sections use oat/manilla.
  Never use pure white or cool-gray surfaces.
- No box shadows, gradients, backdrop blur or glass effects. Elevation comes
  from a surface-tone change and a 1px border.
- Shared cards and overlays use 24px radius. Outlined controls use 12px.
  Filled decisive actions have an 8px **bottom-only** radius.
- `primary` is clay and is reserved for the single most consequential action.
  It turns clay-deep on hover. Destructive confirmation remains dark ink so
  that clay does not become decorative status colour.
- UI chrome (buttons, inputs, labels, table controls) is sans and compact.
  Reading copy, empty states and explanatory messages use serif.
- Generic badges are plain weighted inline text; domain category and production
  markers retain their dot/symbol because their colour is product data and may
  never be the only status signal.
- Keep a 4px spacing rhythm. Use 44px hit targets on phones.

## Responsive constraints

Verify all changes at 375×812, 768×1024, 1280×800 and 1440×900. Dense order
matrices preserve their axes; they do not become generic card grids.
