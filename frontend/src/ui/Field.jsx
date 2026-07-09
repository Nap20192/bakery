const controlClass =
  'w-full rounded-md border border-stone-300 bg-white px-2.5 text-[14px] text-stone-900 transition min-h-11 sm:min-h-9 focus:border-stone-900 focus:outline-none focus:ring-2 focus:ring-stone-900/15 disabled:cursor-not-allowed disabled:opacity-50';

const labelClass = 'mb-1 block text-[12px] font-medium text-stone-600';

export function SelectField({ label, value, onChange, children, className = '', ...props }) {
  return (
    <label className={`block ${className}`.trim()}>
      {label && <span className={labelClass}>{label}</span>}
      <select className={controlClass} value={value} onChange={onChange} {...props}>
        {children}
      </select>
    </label>
  );
}

export function InputField({ label, className = '', ...props }) {
  return (
    <label className={`block ${className}`.trim()}>
      {label && <span className={labelClass}>{label}</span>}
      <input className={controlClass} {...props} />
    </label>
  );
}
