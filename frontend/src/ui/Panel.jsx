// Panel — shadcn/ui Card на семантических токенах. panelClass применяется к
// контейнеру напрямую (исторический API); PanelHeader — шапка с надзаголовком
// и счётчиком позиций.
export const panelClass = 'rounded-xl border border-border bg-card p-3 text-card-foreground shadow-sm sm:p-4';

export function PanelHeader({ eyebrow, title, count }) {
  return (
    <div className="mb-2.5 flex items-start justify-between gap-3">
      <div className="min-w-0">
        {eyebrow && <span className="text-caption font-medium uppercase text-muted-foreground">{eyebrow}</span>}
        <h3 className="m-0 break-words text-title font-semibold text-foreground sm:text-page">{title}</h3>
      </div>
      {count !== undefined && (
        <span className="shrink-0 rounded-full border border-border bg-muted px-2 py-1 text-caption font-medium tabular-nums text-muted-foreground">
          {count} поз.
        </span>
      )}
    </div>
  );
}
