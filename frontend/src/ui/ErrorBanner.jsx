import { cn } from '../lib/cn';

// ErrorBanner — shadcn Alert в destructive-тоне. Renders nothing when there
// is no error text.
export function ErrorBanner({ error, className = '', ...props }) {
  if (!error) return null;
  return (
    <div
      role="alert"
      className={cn('rounded-md border border-destructive/30 bg-destructive/10 px-3 py-2 text-[13px] text-destructive', className)}
      {...props}
    >
      {error}
    </div>
  );
}
