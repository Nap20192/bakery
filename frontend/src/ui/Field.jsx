import { cn } from '../lib/cn';

// Поля формы в стиле shadcn/ui Input/Select + Label, на семантических
// токенах. select остаётся нативным — на телефоне системный пикер удобнее.
export const controlClass =
  'ui-control';

// Compact native control for dense tables and admin toolbars. It deliberately
// keeps sizing separate from the base mobile-first form control.
export const compactControlClass =
  'ui-control ui-control--compact';

const labelClass = 'ui-field__label';

export function SelectField({ label, value, onChange, children, className = '', ...props }) {
  return (
    <label className={cn('ui-field', className)}>
      {label && <span className={labelClass}>{label}</span>}
      <select className={controlClass} value={value} onChange={onChange} {...props}>
        {children}
      </select>
    </label>
  );
}

export function InputField({ label, className = '', ...props }) {
  return (
    <label className={cn('ui-field', className)}>
      {label && <span className={labelClass}>{label}</span>}
      <input className={controlClass} {...props} />
    </label>
  );
}
