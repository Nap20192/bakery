export function MetaCell({ label, value }) {
  return (
    <div className="min-w-0 rounded-md border border-stone-200 bg-stone-50 px-2.5 py-1.5 sm:px-3">
      <span className="block text-[11px] leading-4 text-stone-500">{label}</span>
      <strong className="block break-words text-[12px] font-medium leading-5 text-stone-900 sm:text-[13px]">{value}</strong>
    </div>
  );
}
