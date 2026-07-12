import { categoryStyle } from '../lib/categories';

// CategoryBadge is the colored тип-заявки pill shown on order cards and
// details. Renders nothing for orders without a category (legacy orders).
export function CategoryBadge({ category, className = '' }) {
  if (!category?.name) return null;
  const style = categoryStyle(category);
  return (
    <span
      className={`inline-flex shrink-0 items-center gap-1 text-caption font-semibold leading-4 text-foreground ${className}`.trim()}
    >
      <span className={`h-1.5 w-1.5 rounded-full ${style.dot}`} aria-hidden="true" />
      {category.name}
    </span>
  );
}
