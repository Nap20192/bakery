import { Icon } from '../../ui/Icon';
import { categoryStyle } from '../../lib/categories';
import { orderCreator } from '../../lib/orders';

// ProductionMark показывает, что по заказу есть отклонения отработки
// (в журнале хранятся только позиции, где факт отличается от заявки).
function ProductionMark({ items = [] }) {
  const deviations = items.filter((item) => item.produced_quantity != null).length;
  if (deviations === 0) return null;
  return (
    <span className="shrink-0 text-[11px] font-semibold leading-5 text-amber-600" title="Есть отклонения по отработке">
      ±{deviations}
    </span>
  );
}

// BakerOrderCard is a compact matrix cell card: tap opens the quick-look
// popup (or toggles the checkbox in selection mode). The card carries the
// position count and the author — shop, date and тип are given by the matrix
// axes and the stripe color; the number lives in the popup.
export function BakerOrderCard({ order, selected, checked, selectionMode, onToggleSelection, onPreview }) {
  const handleClick = selectionMode ? onToggleSelection : onPreview;
  const style = order.category ? categoryStyle(order.category) : null;

  return (
    <button
      type="button"
      onClick={handleClick}
      className={`group relative w-full rounded-lg border bg-white px-2 py-1.5 text-left transition duration-150 focus:outline-none focus:ring-2 focus:ring-stone-900/20 active:scale-[0.98] ${
        style ? `border-l-4 ${style.stripe}` : ''
      } ${
        checked
          ? 'border-stone-900 ring-1 ring-stone-900/15'
          : selected && !selectionMode
            ? 'border-stone-800 shadow-sm'
            : 'border-stone-200 hover:border-stone-400 hover:shadow-sm'
      }`}
    >
      <span className="flex items-baseline justify-between gap-1">
        <strong className="min-w-0 truncate text-[13px] font-semibold leading-5 text-stone-950">
          {order.items?.length || 0} поз.
        </strong>
        {order.cancelled ? (
          <span className="shrink-0 text-[11px] font-semibold uppercase leading-5 text-red-600">Отм.</span>
        ) : (
          <ProductionMark items={order.items} />
        )}
      </span>
      <span className="mt-0.5 block truncate text-[11px] leading-4 text-stone-500">{orderCreator(order)}</span>
      {selectionMode && (
        <span
          className={`absolute -right-1 -top-1 inline-flex h-5 w-5 items-center justify-center rounded-full border transition duration-150 ${
            checked ? 'border-stone-900 bg-stone-900 text-white' : 'border-stone-300 bg-white text-transparent group-hover:border-stone-400'
          }`}
          aria-hidden="true"
        >
          <Icon name="select" size={11} />
        </span>
      )}
    </button>
  );
}
