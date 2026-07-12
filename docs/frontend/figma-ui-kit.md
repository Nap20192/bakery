# Figma UI kit reference

Bakery uses the following Figma Community file as its canonical external UI
kit. It is the source of truth for colors, typography, layout, grids, spacing,
buttons, tags, fields, tables, overlays, states and responsive composition:

- [MCP Apps for Claude (Community)](https://www.figma.com/design/zcWlfgVfMNX576XBSC99Z6/MCP-Apps-for-Claude--Community-?node-id=467-20292&p=f&t=fTfT3Zbz8Lc0YnU8-0)
- file key: `zcWlfgVfMNX576XBSC99Z6`
- verified node: `467:20292` (`🖌️ Color`)
- verified frame: `467:20293` (`Color`, 9208×1509)
- verified typography node: `7:19` (`Font`)
- verified border node: `467:21770` (`Border`)

## How agents use it

1. Read `.interface-design/system.md` before visible frontend changes.
2. For a supplied Figma URL, extract its file key and exact node ID.
3. Use Figma MCP `get_design_context` for implementation context. If the
   desktop-plugin selection is unavailable, use `get_screenshot` and
   `get_metadata` to inspect the node, then ask for a selected component only
   when exact variables or generated reference code are required.
4. Search the Figma design system before creating a new pattern.
5. Map exact Figma values into Bakery semantic CSS tokens; never paste raw
   values across feature files.
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
`color-text-danger` `#7F2C28`, and light danger `#EE8884`. They are mapped to
Bakery's shared semantic tokens. Other exact values require an active layer
selection in Figma Desktop; do not introduce a new visual token until its
Figma node has been read.

## Verified typography and border structure

Node `7:19` defines two families: `Anthropic Sans` for interface copy and
`JetBrains Mono` for technical values. Its weight tokens are Normal (400),
Medium (500), Semibold (600), and Bold (700); its text roles are
XS/SM/MD/LG/XL/2XL/3XL. Bakery maps them to the semantic type scale in
`src/styles.css`; Inter is bundled as an explicit fallback because the kit's
Anthropic Sans asset is not published through npm and Bakery's UI is Russian.

Node `467:21770` defines the border vocabulary: radius
XS/SM/MD/LG/XL/Full and a regular border width. Shared components keep their
radius values in semantic tokens, and use `Full` only for tags and compact
status markers.

Node `467:21815` contains separate desktop-web and mobile icon sets. Bakery
keeps its existing semantic SVG names, but applies the kit's lighter outlined
stroke as the default. A feature may request a heavier stroke only where the
action needs explicit visual emphasis.

## Project-specific constraints

- Bakery category colors are domain data and stay synchronized with backend
  `CategoryColors`.
- Status cannot depend on color alone: production uses `✓`/`±`, cancellation
  uses `✕` and text treatment.
- Dense operational tables and production editors take priority over generic
  card layouts.
- Existing API, roles and production persistence rules are unaffected by this
  visual reference.
