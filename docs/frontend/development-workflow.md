# Frontend Development Workflow

The Bakery frontend is a server-rendered Go application in `frontend/`.
`html/template` renders full pages, HTMX 2.0.4 progressively enhances links
and forms, `static/app.css` contains the design system, and `static/app.js`
contains the small amount of client state that HTML cannot express directly.

Templates are split by role:

```
templates/layout.html       the shell
templates/pages/            one file per page.View, template name == View
templates/components/       partials shared between pages
```

`templates.go` owns the parse step and the FuncMap. Two helpers exist because
`{{template}}` cannot express them: `view` dispatches on `page.View` at run time
(the layout would otherwise need an if-chain per page), and `dict` builds the
argument map for a component (`{{template}}` passes a single value). A key left
out of `dict` renders as its zero value, so optional component parameters are
simply omitted; `isset` tells an omitted parameter apart from a zero count.

## Architecture rules

| Area | Bakery |
|---|---|
| Runtime | Go binary: `go run ./frontend` |
| Rendering | standard-library `html/template` + embedded templates |
| Browser interaction | vendored HTMX 2.0.4 + plain JavaScript |
| Styling | plain mobile-first CSS with semantic tokens |
| API access | `application.Queries` / `application.Commands` ports, implemented by `frontend/internal/backend`; browser never calls JSON API |
| Authentication | API credential in `HttpOnly`, `SameSite=Lax` cookie |
| Contract | `docs/api/openapi.yaml` + compiled Go client/tests |
| Browser QA | Playwright and axe |

The frontend is a BFF, not a second backend. It owns presentation, cookies,
HTML routes, CSRF protection, and view-model projection. Business validation,
authorization, persistence, and calculations remain in the worker API.

## Required loop

1. Read `FRONTEND_BEHAVIOR.md`, the relevant handler/template, `app.css`,
   `app.js`, `DESIGN.md`, and `.interface-design/system.md`.
2. Confirm the API operation in `docs/api/openapi.yaml` and
   `internal/inbound/api/contract` and the relevant application gateway.
3. State the user, primary action, information hierarchy, phone behavior, and
   accessibility states before editing visible UI.
4. Implement one vertical flow: contract -> backend adapter -> Query or
   Command -> web handler -> view model -> template -> CSS/JS enhancement.
5. Run `gofmt`, `go test ./frontend/...`, `go vet ./frontend/...`, then the
   repository checks for any backend changes.
6. Start worker and frontend, then exercise the real flow at 375x812,
   768x1024, 1280x800, and 1440x900.
7. Check keyboard focus, native dialogs, horizontal table overflow, long
   Russian values, loading/error/success states, console, network, and axe.
8. Review `git diff --check`, `git diff --stat`, and the focused diff.

## HTMX conventions

- `#app-shell` is the boosted navigation boundary. Full responses still work
  without HTMX; HTMX selects and swaps only `#app-shell` and pushes history.
- Small server fragments live under `/fragments/*` and render named templates.
- Forms use ordinary `action` and `method`. HTMX adds request disabling,
  indicators, and targeted swaps; it does not replace server validation.
- API failures keep their HTTP status. `app.js` allows HTMX to swap safe HTML
  error responses instead of hiding them.
- Mutating requests require the CSRF cookie plus `_csrf` form value or
  `X-CSRF-Token`. The script adds the header for HTMX requests.
- Use `HX-Location` after successful mutations so the server owns the next
  canonical URL.
- Do not add inline scripts or `hx-on` JavaScript. The CSP permits local
  assets and the official Telegram bridge only.

## JavaScript boundary

Plain JavaScript is allowed for Telegram initData exchange, mobile navigation,
native dialog lifecycle, local batch selection, catalog filtering, production
dirty-state enforcement, and small ergonomic input relationships. New business
rules do not belong in `app.js`.

## Responsive and accessibility contract

- Controls are at least 44px on phones and form controls render at 16px to
  avoid iOS viewport zoom.
- Operational matrices remain tables. On narrow screens, identifying axes are
  sticky and the data plane scrolls horizontally.
- Category and production colors always include text or a symbol.
- Every icon-only action has an accessible name; destructive actions use the
  native `<dialog>` confirmation flow.
- Motion is optional and disabled under `prefers-reduced-motion`.

## Commands

```bash
# terminal 1
go run ./cmd/worker

# terminal 2
FRONTEND_ADDR=:5173 BACKEND_URL=http://127.0.0.1:8080 go run ./frontend

# static checks
gofmt -w frontend/main.go frontend/internal
go test ./frontend/...
go vet ./frontend/...
```
