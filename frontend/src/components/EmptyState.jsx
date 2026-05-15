export function EmptyState({ children, compact = false }) {
  return (
    <div className={`rounded-md border border-dashed border-stone-300 bg-stone-50 text-center text-[13px] text-stone-500 ${compact ? 'p-3' : 'p-6'}`}>
      {children}
    </div>
  );
}
