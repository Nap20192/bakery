# Techcard Service

`internal/services/techcard/` — builds a human-readable tech card (техкарта)
for a product code from the stored iiko snapshot.

## Layout

```
domain/             TechCard model
usecase/techcard/   package techcarduc — UseCase + Repository
infra/repo/         techcardrepo — assembles the card from snapshot tables
app/app.go          techcardapp.New(queries)
```

No HTTP adapter. The service is wired into the worker's `AppDeps`
(`WithTechCardService`) as an internal capability; a future delivery adapter
should be admin-gated.

## Contract

```go
type UseCase interface {
    GetByCode(ctx context.Context, code string, date time.Time) (techcarddomain.TechCard, error)
}
```

`date` selects the chart valid on that day (iiko charts are dated); the
repository resolves the product by code and expands its recipe rows from the
snapshot.

## Relation to monitor

Techcard renders **one product's** card for reading; monitor walks the
**whole graph** recursively to compute ingredient totals. Both read the same
snapshot tables populated by [sync](sync.md) and must stay consistent with
the assembly/prepared formulas described in
[monitor.md](monitor.md#calculation).
