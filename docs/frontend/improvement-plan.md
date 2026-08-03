# Frontend Improvement TODO

The July 2026 rewrite replaced the React/Vite/Tailwind SPA with a Go BFF,
server-rendered templates, vendored HTMX, plain CSS, and plain JavaScript.

## Completed in the rewrite

- [x] Removed Node, Vite, React, Tailwind, Radix, openapi-fetch, and generated
  TypeScript schema from the runtime and build.
- [x] Moved API credentials out of browser storage into an `HttpOnly` cookie.
- [x] Added CSRF protection, CSP, safe redirects, request logging, and graceful
  shutdown to the frontend service.
- [x] Preserved role-specific routes, the baker matrix, summary table,
  production journal/editor, dough calculator, and admin workflows.
- [x] Kept load-bearing tables as tables with sticky axes on phones.
- [x] Added typed template smoke tests for every view.

## Next checks

- [ ] Add repeatable Playwright + axe scenarios for shop, baker, and admin.
- [ ] Add screenshot baselines for 375x812, 768x1024, 1280x800, and 1440x900.
- [ ] Add API-client contract fixtures for representative OpenAPI responses.
- [ ] Profile production-journal request fan-out; batch API endpoints are
  preferable if the number of sheets makes first-order category lookup slow.
