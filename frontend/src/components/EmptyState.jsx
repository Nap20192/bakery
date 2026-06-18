export function EmptyState({ children, compact = false }) {
  return (
    <div className={`rounded-md border border-dashed border-stone-300 bg-white text-center text-[13px] text-stone-600 ${compact ? 'p-3' : 'p-6'}`}>
      {children}
    </div>
  );
}
