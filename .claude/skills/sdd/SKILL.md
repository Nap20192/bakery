---
name: sdd
description: Spec-driven development for this repo — write or change the specification before the code, keep specs and code reconciled, and audit a spec for drift against the implementation. Use when adding or changing a feature, when asked to update/write a specification, when a spec and the code disagree, or when starting work whose behavior is described in docs/ or FRONTEND_BEHAVIOR.md.
---

# Spec-driven development

The specification is the source of truth for **behavior**. The code is the source
of truth for **implementation**. When they disagree, that is a defect in one of
them — never something to leave alone.

This skill covers the workflow. For how to structure a document (tutorial vs
how-to vs reference vs explanation), use the `documentation-writer` skill.

## Where the specs live

| Document | Owns |
|---|---|
| `frontend/FRONTEND_BEHAVIOR.md` | Frontend routes, per-role visibility, interaction rules, copy and layout contracts |
| `docs/constraints.md` | Numbered cross-cutting constraints (C1, C2, …) referenced from other specs |
| `docs/services/<svc>.md` | Per-service domain rules and invariants |
| `docs/architecture.md` | Layering, dependency direction, composition |
| `docs/database.md` | Schema and migration rules |
| `docs/api/openapi.yaml` | HTTP contract. The route-sync test fails if a route is missing |
| `docs/edge-cases.md` | Agreed behavior at boundaries |
| `docs/frontend/development-workflow.md` | How to build frontend changes |
| `CLAUDE.md` / `AGENTS.md` | Instructions to agents, not user-facing behavior |

`docs/api/openapi.yaml` is the only spec with an automated guard
(`internal/inbound/api/openapi_test.go`). Every other document drifts silently,
so it has to be checked by hand.

## The loop

1. **Read the spec first.** Find the section that owns the behavior you are
   about to change. If no section owns it, decide where it belongs before
   writing code — that decision is part of the design.
2. **Change the spec when the intended behavior changes.** A new rule, a
   different default, a removed affordance. Do this in the same change as the
   code, not afterwards.
3. **Implement.**
4. **Reconcile.** Re-read the section you touched and the sections next to it.
   Behavior changes leak: removing a filter reset also invalidates the sentence
   about filters three sections down.
5. **Verify the claims you wrote**, using the audit below.

Do not update the spec to match a bug. If the code disagrees with a rule that
still makes sense, fix the code.

## Auditing a spec for drift

Prose is not evidence. Before trusting or rewriting a claim, check it:

- Quote the claim, then grep for the thing it names — the label, the CSS class,
  the route, the field. A claim about a button says the button's text; if that
  string is not in the templates, the claim is stale.
- Prefer checking the rendered result over the source when the claim is about
  what the user sees.
- Claims that go stale most often: control labels, which roles see what,
  "popup" vs "page", default values, and any sentence that begins "при
  изменении …" describing a side effect.

Report contradictions instead of quietly rewriting them, and say which side you
changed and why.

## What belongs in a spec

**Yes** — observable behavior and the reason for it: what each role can do, what
a control does, what happens at boundaries, invariants that must not regress,
and constraints that cost something to rediscover.

**No** — file layout, function names, framework mechanics, or anything a reader
can get from the code faster than from prose. Implementation notes belong in the
workflow docs or a code comment next to the thing they explain.

Record the **why** for any rule that looks arbitrary. A rule without a reason
gets deleted by the next person who finds it inconvenient.

## Constraints

Cross-cutting rules get a number in `docs/constraints.md` (C1, C2, …) and are
referenced by number from the specs that depend on them. Add a new number rather
than restating the rule in two places.
