// Category color palette. The backend stores a color slug per тип заявки;
// this map turns it into concrete Tailwind classes (literal strings so the
// JIT compiler can see them). Keep in sync with domain.CategoryColors.
export const CATEGORY_COLORS = {
  amber: {
    stripe: 'border-l-amber-400',
    badge: 'border-amber-200 bg-amber-50 text-amber-800',
    dot: 'bg-amber-400',
    chipActive: 'border-amber-400 bg-amber-50 text-amber-900',
    pick: 'border-amber-300 bg-amber-50 hover:border-amber-400',
  },
  sky: {
    stripe: 'border-l-sky-400',
    badge: 'border-sky-200 bg-sky-50 text-sky-800',
    dot: 'bg-sky-400',
    chipActive: 'border-sky-400 bg-sky-50 text-sky-900',
    pick: 'border-sky-300 bg-sky-50 hover:border-sky-400',
  },
  violet: {
    stripe: 'border-l-violet-400',
    badge: 'border-violet-200 bg-violet-50 text-violet-800',
    dot: 'bg-violet-400',
    chipActive: 'border-violet-400 bg-violet-50 text-violet-900',
    pick: 'border-violet-300 bg-violet-50 hover:border-violet-400',
  },
  emerald: {
    stripe: 'border-l-emerald-400',
    badge: 'border-emerald-200 bg-emerald-50 text-emerald-800',
    dot: 'bg-emerald-400',
    chipActive: 'border-emerald-400 bg-emerald-50 text-emerald-900',
    pick: 'border-emerald-300 bg-emerald-50 hover:border-emerald-400',
  },
  rose: {
    stripe: 'border-l-rose-400',
    badge: 'border-rose-200 bg-rose-50 text-rose-800',
    dot: 'bg-rose-400',
    chipActive: 'border-rose-400 bg-rose-50 text-rose-900',
    pick: 'border-rose-300 bg-rose-50 hover:border-rose-400',
  },
  stone: {
    stripe: 'border-l-stone-400',
    badge: 'border-stone-300 bg-stone-100 text-stone-700',
    dot: 'bg-stone-400',
    chipActive: 'border-stone-500 bg-stone-100 text-stone-900',
    pick: 'border-stone-300 bg-stone-50 hover:border-stone-400',
  },
};

export const CATEGORY_COLOR_SLUGS = Object.keys(CATEGORY_COLORS);

export function categoryStyle(category) {
  return CATEGORY_COLORS[category?.color] || CATEGORY_COLORS.stone;
}
