import { cva } from 'class-variance-authority';
import { cn } from '../lib/cn';

// Button — shadcn/ui-кнопка на cva и семантических токенах. Имена вариантов
// исторические (см. вызовы по фичам): default = outline, primary = solid
// primary, danger = outline destructive, ghost — прозрачная.
const buttonVariants = cva(
  'inline-flex min-h-11 items-center justify-center gap-1.5 whitespace-nowrap rounded-md px-3 py-1.5 text-[13px] font-semibold transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring/30 disabled:pointer-events-none disabled:opacity-50 sm:min-h-9',
  {
    variants: {
      variant: {
        default: 'border border-input bg-card text-foreground shadow-xs hover:bg-accent hover:text-accent-foreground',
        primary: 'bg-primary text-primary-foreground shadow-xs hover:bg-primary/90',
        danger:
          'border border-destructive/40 bg-card text-destructive shadow-xs hover:border-destructive/60 hover:bg-destructive/10 focus-visible:ring-destructive/30',
        ghost: 'text-muted-foreground hover:bg-accent hover:text-foreground',
      },
    },
    defaultVariants: { variant: 'default' },
  },
);

export function Button({ variant, className = '', ...props }) {
  return <button className={cn(buttonVariants({ variant }), className)} {...props} />;
}
