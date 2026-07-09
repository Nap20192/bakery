// ErrorBanner is the shared inline error message block. Renders nothing when
// there is no error text.
export function ErrorBanner({ error, className = '' }) {
  if (!error) return null;
  return (
    <div role="alert" className={`rounded-md border border-red-200 bg-red-50 px-3 py-2 text-[13px] text-red-800 ${className}`.trim()}>
      {error}
    </div>
  );
}
