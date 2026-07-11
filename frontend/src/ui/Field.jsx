import { cn } from '../lib/cn';

// Поля формы в стиле shadcn/ui Input/Select + Label, на семантических
// токенах. select остаётся нативным — на телефоне системный пикер удобнее.
export const controlClass =
  'flex min-h-11 w-full rounded-md border border-input bg-card px-2.5 text-[14px] text-foreground shadow-xs transition-colors placeholder:text-muted-foreground focus-visible:border-ring focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring/20 disabled:cursor-not-allowed disabled:opacity-50 sm:min-h-9';

const labelClass = 'mb-1 block text-[12px] font-medium text-muted-foreground';

export function SelectField({ label, value, onChange, children, className = '', ...props }) {
  return (
    <label className={cn('block', className)}>
      {label && <span className={labelClass}>{label}</span>}
      <select className={controlClass} value={value} onChange={onChange} {...props}>
        {children}
      </select>
    </label>
  );
}

export function InputField({ label, className = '', ...props }) {
  return (
    <label className={cn('block', className)}>
      {label && <span className={labelClass}>{label}</span>}
      <input className={controlClass} {...props} />
    </label>
  );
}
