const baseClass =
  'inline-flex min-h-11 items-center justify-center gap-1.5 rounded-md border px-3 py-1.5 text-[13px] font-semibold transition focus:outline-none focus:ring-2 focus:ring-stone-900/20 disabled:cursor-not-allowed disabled:opacity-50 sm:min-h-9';

const variants = {
  default: `${baseClass} border-stone-300 bg-white text-stone-800 hover:border-stone-400 hover:bg-stone-50`,
  primary: `${baseClass} border-stone-900 bg-stone-900 text-white hover:bg-stone-800`,
};

export function Button({ variant = 'default', className = '', ...props }) {
  return <button className={`${variants[variant]} ${className}`.trim()} {...props} />;
}
