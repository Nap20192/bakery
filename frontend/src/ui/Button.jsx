import { cn } from '../lib/cn';

// Button uses plain component CSS and semantic tokens. Variant names remain
// исторические (см. вызовы по фичам): default = outline, primary = solid
// primary, danger = outline destructive, ghost — прозрачная.
export function Button({ variant, className = '', loading = false, disabled = false, children, ...props }) {
  const variantClass = `ui-button--${variant || 'default'}`;
  return (
    <button
      className={cn('ui-button', variantClass, className)}
      disabled={loading || disabled}
      aria-busy={loading || undefined}
      {...props}
    >
      {loading && <LoadingIndicator />}
      {children}
    </button>
  );
}

function LoadingIndicator() {
  return (
    <svg className="ui-button__spinner" viewBox="0 0 24 24" fill="none" aria-hidden="true">
      <circle className="ui-button__spinner-track" cx="12" cy="12" r="9" stroke="currentColor" strokeWidth="3" />
      <path className="ui-button__spinner-head" d="M21 12a9 9 0 0 0-9-9" stroke="currentColor" strokeWidth="3" strokeLinecap="round" />
    </svg>
  );
}
