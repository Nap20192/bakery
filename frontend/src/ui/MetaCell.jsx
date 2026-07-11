export function MetaCell({ label, value }) {
  return (
    <div className="min-w-0 rounded-md border border-border bg-muted px-2.5 py-1.5 sm:px-3">
      <span className="block text-[11px] leading-4 text-muted-foreground">{label}</span>
      <strong className="block break-words text-[12px] font-medium leading-5 text-foreground sm:text-[13px]">{value}</strong>
    </div>
  );
}
