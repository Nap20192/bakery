export function MetaCell({ label, value }) {
  return (
    <div className="min-w-0 rounded-md border border-border bg-muted px-2.5 py-1.5 sm:px-3">
      <span className="block text-caption text-muted-foreground">{label}</span>
      <strong className="block break-words text-body font-medium text-foreground">{value}</strong>
    </div>
  );
}
