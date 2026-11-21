# bakery: native Review and implementation Tasks

Review and commit this entire authority bundle before planning. The common committed
`.af/task-catalog.toml` pins current `pipeline.toml` and `worker.toml` packages;
`.af/code-policy.toml` declares the acceptance checks. No Campaign configuration is emitted.

## First Review

Copy `.af/task-packages/onboard/review-code/task.json` to `../review.json` and give it a fresh
Task ID. Make a source change, then run:

```sh
af review plan --file ../review.json --authority HEAD --base HEAD --uncommitted
af review resume review --confirm-plan PLAN_ID
af review inspect review --json
af review ledger review --json
af review report review --json
```

Planning records a Task without dispatching Workers. Inspect its exact plan before approving
execution. `--authority` selects trusted committed policy independently of `--candidate REV`
and `--base REV`; `--uncommitted` selects the working tree instead of a candidate commit.
Empty Diff planning and execution refuse before checks, Provider admission or Workers.
Use the catalog's machine-local Provider aliases, or a `--bindings FILE` document with
`[providers]` entries keyed by `onboard/bugs` and `onboard/correctness`. `af provider detect`
and `af provider status` are token-free; login runs at the operator's terminal.

The example's `strategy` is `light`: one complete Round. Heavy review is an explicit native
Pipeline with bounded successor Rounds and captured convergence policy; there is no `--heavy`
Campaign switch. See `af help review` and the Review Tasks guide for native composition.
Resume a recorded Task with `af review resume review --confirm-plan PLAN_ID`.
Continue reviewed implementation work with `af review continue NEXT_ID --from TASK_ID` using
its captured continuation contract. These commands never reset prior accounting or authority.

## Model-driven Tasks

`[task.defaults]` selects `onboard/implementation-reviewed` and its bounded independent
implementation, review and evaluation. A three-field Task file can use those defaults with
`af task plan --file ticket.json`. Review the plan before running it. A sealed Proposal may be
exported with `af review export TASK_ID PROPOSAL_ID`; export grants no delivery or publication
authority. Branches, commits, pushes and pull requests remain separate human actions.

Existing authority is validated by `af onboard` and `af doctor`. To re-pin intentional current
package edits, run `af onboard --refresh-lock`, inspect the catalog diff and commit it. No
Campaign reader or migration is installed; replace retired `.af/pipelines/` configuration
with the common Task catalog after reviewing a fresh scaffold in a separate checkout.
