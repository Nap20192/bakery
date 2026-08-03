# Figma UI kit reference

Bakery uses the following Figma Community file as a secondary external UI
reference. `DESIGN.md` is the source of truth for colors, typography, spacing,
surfaces and component styling. Use Figma for component anatomy, iconography,
layout ideas and exact node inspection when a task explicitly references a
Figma frame.

- [MCP Apps for Claude (Community)](https://www.figma.com/design/zcWlfgVfMNX576XBSC99Z6/MCP-Apps-for-Claude--Community-?node-id=467-20292&p=f&t=fTfT3Zbz8Lc0YnU8-0)
- file key: `zcWlfgVfMNX576XBSC99Z6`
- verified node: `467:20292` (`🖌️ Color`)
- verified frame: `467:20293` (`Color`, 9208×1509)
- verified typography node: `7:19` (`Font`)
- verified border node: `467:21770` (`Border`)

## How agents use it

1. Read `DESIGN.md` and `.interface-design/system.md` before visible frontend changes.
2. For a supplied Figma URL, extract its file key and exact node ID.
3. Use Figma MCP `get_design_context` for implementation context. If the
   desktop-plugin selection is unavailable, use `get_screenshot` and
   `get_metadata` to inspect the node, then ask for a selected component only
   when exact variables or generated reference code are required.
4. Search the Figma design system before creating a new pattern when the task
   is Figma-driven.
5. Map approved Figma structure into Bakery semantic CSS tokens; never paste
   raw values across feature files.
6. Adapt Figma components to Bakery's roles and content; do not copy demo
   screens verbatim.

## Verified color structure

Node `467:20292` contains eight groups:

| Figma group | Bakery purpose |
|---|---|
| Background / Surface | page, panel and inset surfaces |
| Background / Accent | selected and semantic accents |
| Text / Surface | primary through muted text hierarchy |
| Text / Accent | readable text on semantic accents |
| Border / Surface | quiet layout and control boundaries |
| Border / Accent | selected, warning and status boundaries |
| Ring / Surface | keyboard focus on neutral surfaces |
| Ring / Accent | keyboard focus on accented surfaces |

The public Figma embed verifies these baseline values from the Color canvas:
`color-text-info` `#3266B5`, light info/border `#80AADD`,
`color-text-danger` `#7F2C28`, and light danger `#EE8884`. These values are
documented as observed Figma values only; they are not Bakery's active palette
unless a future task explicitly changes `DESIGN.md`. Other exact values require
an active layer selection in Figma Desktop.

## Verified typography and border structure

Node `7:19` defines `Anthropic Sans` for interface copy and `JetBrains Mono`
for technical values. Bakery now follows `DESIGN.md`: serif headings,
humanist sans body/UI and mono technical values. System fonts are used as
local fallbacks; no font CDN is required.

Node `467:21770` defines the border vocabulary: radius
XS/SM/MD/LG/XL/Full and a regular border width. Shared components keep their
radius values in semantic tokens, and use `Full` only for tags and compact
status markers.

Node `467:21815` contains separate desktop-web and mobile icon sets. The
server-rendered UI prefers labelled controls and familiar text symbols. When a
new icon is necessary, add it once as an embedded static asset with an
accessible label; do not copy ad-hoc SVG into page templates.

## Project-specific constraints

- Bakery category colors are domain data and stay synchronized with backend
  `CategoryColors`.
- Status cannot depend on color alone: production uses `✓`/`±`, cancellation
  uses `✕` and text treatment.
- Dense operational tables and production editors take priority over generic
  card layouts.
- Existing API, roles and production persistence rules are unaffected by this
  visual reference.
