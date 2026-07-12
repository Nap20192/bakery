import { cn } from '../lib/cn';

export function EmptyState({ children, compact = false }) {
  return (
    <div
      className={cn(
        'ui-empty-state',
        compact && 'ui-empty-state--compact',
      )}
    >
      {children}
    </div>
  );
}
