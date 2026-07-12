import { cn } from '../lib/cn';

// ErrorBanner — shadcn Alert в destructive-тоне. Renders nothing when there
// is no error text.
export function ErrorBanner({ error, className = '', ...props }) {
  if (!error) return null;
  return (
    <div
      role="alert"
      className={cn('ui-error-banner', className)}
      {...props}
    >
      {error}
    </div>
  );
}
