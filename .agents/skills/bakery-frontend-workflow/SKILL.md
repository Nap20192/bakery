---
name: bakery-frontend-workflow
description: End-to-end workflow for implementing, reviewing, or refining the Bakery Go/HTMX frontend. Use for tasks that edit frontend/, HTML behavior, handlers, templates, CSS, JavaScript, OpenAPI-facing client code, responsive layout, accessibility, or browser interactions.
---

# Bakery Frontend Workflow

Follow `AGENTS.md` and read `docs/frontend/development-workflow.md` before editing.

## 1. Brief and research

Restate goal, context, constraints, and done criteria. Inspect the relevant
handler, template, tokens in `static/app.css`, API client, route, behavior
spec, and similar UI. Do not add a pattern before searching for an equivalent.

Ask one concise guiding question only when the answer materially changes
business rules, persistence/API shape, destructive behavior, or visual
direction. Otherwise state the smallest repo-consistent assumption.

## 2. Design direction

For visible UI changes use `interface-design`, then `typeui-fundamentals`.
State user, task, feel, hierarchy, responsive behavior, and accessibility.
Preserve the editorial serif/sans hierarchy, flour/crust tokens,
category/date semantics, and UI-kit.
Do not create a parallel design system or generic SaaS card layout.

## 3. Plan and implement

List files to modify. Deliver one complete vertical flow at a time. Keep reads
and writes behind `frontend/internal/application` Queries and Commands, worker
HTTP calls in `frontend/internal/backend`, projection in web handlers/view
models, reusable markup in named templates, tokens in `static/app.css`, and
business rules on the worker backend. Shared JSON DTO changes belong in
`internal/inbound/api/contract` together with the OpenAPI update.

## 4. Static validation

Run the checks relevant to the diff:

```bash
gofmt -w frontend/main.go frontend/internal
go test ./frontend/...
go vet ./frontend/...
go build -o /tmp/bakery-frontend ./frontend
```

For contract/backend changes also run `make sqlc` when needed,
`go test ./internal/inbound/api`, `make frontend-check`, and relevant Go tests.

## 5. Browser loop

Use `playwright-interactive` for exploratory verification and `playwright` for
repeatable tests. Start worker and the Go frontend, then test the complete flow at 375×812,
768×1024, 1280×800, and 1440×900. Check keyboard focus, overflow, sticky/fixed
elements, long Russian content, loading/empty/error/success states, console
errors, and failed requests. Capture screenshots for visible changes. Fix issues
and repeat; build success alone is not visual verification.

If browser skills or dependencies are unavailable, report that limitation
explicitly and provide the exact setup command. Never claim browser validation
without evidence.

## 6. Accessibility and review

Run axe where configured, then manually verify labels, focus order, icon names,
color-independent meaning, touch targets, and reduced motion. Review only the
intended diff, preserve unrelated dirty-worktree changes, and exclude
`cmd/testingorders/main.go` from commits.

## 7. Completion report

Report files changed, decisions, exact commands and results, browser viewports,
console/network findings, accessibility result, and remaining limitations.
