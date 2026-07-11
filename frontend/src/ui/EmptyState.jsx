import { cn } from '../lib/cn';

export function EmptyState({ children, compact = false }) {
  return (
    <div
      className={cn(
        'rounded-md border border-dashed border-input bg-card text-center text-[13px] text-muted-foreground',
        compact ? 'p-3' : 'p-6',
      )}
    >
      {children}
    </div>
  );
}
