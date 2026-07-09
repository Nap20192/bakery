import { categoryStyle } from '../lib/categories';

// CategoryBadge is the colored тип-заявки pill shown on order cards and
// details. Renders nothing for orders without a category (legacy orders).
export function CategoryBadge({ category, className = '' }) {
  if (!category?.name) return null;
  const style = categoryStyle(category);
  return (
    <span
      className={`inline-flex shrink-0 items-center gap-1 rounded-full border px-2 py-0.5 text-[11px] font-semibold leading-4 ${style.badge} ${className}`.trim()}
    >
      <span className={`h-1.5 w-1.5 rounded-full ${style.dot}`} aria-hidden="true" />
      {category.name}
    </span>
  );
}
