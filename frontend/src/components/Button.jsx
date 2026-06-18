const baseClass =
  'inline-flex min-h-8 items-center justify-center rounded-md border px-2.5 py-1.5 text-[13px] font-medium transition disabled:cursor-not-allowed disabled:opacity-50';

const variants = {
  default: `${baseClass} border-stone-300 bg-white text-stone-800 hover:border-stone-400 hover:bg-stone-50`,
  primary: `${baseClass} border-stone-900 bg-stone-900 text-white hover:bg-stone-800`,
};

export function Button({ variant = 'default', className = '', ...props }) {
  return <button className={`${variants[variant]} ${className}`.trim()} {...props} />;
}
