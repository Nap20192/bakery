import { Button } from '../../ui/Button';
import { Icon } from '../../ui/Icon';
import { categoryStyle } from '../../lib/categories';

// BakerOrderFilters is intentionally minimal: тип заявки chips + selection
// toggle. Shops and dates are already the matrix axes, so their filters are
// redundant here.
export function BakerOrderFilters({ filters, categories = [], selectionMode, onToggleSelectionMode, onFiltersChange }) {
  return (
    <section className="flex flex-wrap items-center gap-1.5">
      <CategoryChip
        label="Все"
        active={!filters.categoryID}
        onClick={() => onFiltersChange({ categoryID: '' })}
      />
      {categories.map((category) => (
        <CategoryChip
          key={category.id}
          label={category.name}
          category={category}
          active={String(filters.categoryID) === String(category.id)}
          onClick={() => onFiltersChange({ categoryID: String(category.id) })}
        />
      ))}
      <Button variant={selectionMode ? 'primary' : 'default'} onClick={onToggleSelectionMode} className="ml-auto !min-h-9">
        <Icon name="select" size={15} />
        Выбор
      </Button>
    </section>
  );
}

// CategoryChip is a тип-заявки filter toggle colored with the category color.
function CategoryChip({ label, category, active, onClick }) {
  const style = category ? categoryStyle(category) : null;
  const activeClass = style ? style.chipActive : 'border-stone-900 bg-stone-900 text-white';
  return (
    <button
      type="button"
      onClick={onClick}
      className={`inline-flex min-h-9 items-center gap-1.5 rounded-full border px-3 text-[13px] font-medium transition focus:outline-none focus:ring-2 focus:ring-stone-900/20 ${
        active ? activeClass : 'border-stone-300 bg-white text-stone-700 hover:border-stone-400'
      }`}
    >
      {style && <span className={`h-2 w-2 rounded-full ${style.dot}`} aria-hidden="true" />}
      {label}
    </button>
  );
}
