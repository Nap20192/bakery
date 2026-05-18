export function EmptyState({ children, compact = false }) {
  return (
    <div className={`rounded-md border border-dashed border-stone-300 bg-[#fff7df] text-center text-[13px] text-stone-600 ${compact ? 'p-3' : 'p-6'}`}>
      {children}
    </div>
  );
}
