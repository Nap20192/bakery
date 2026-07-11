import { Icon } from '../../ui/Icon';
import { categoryStyle } from '../../lib/categories';
import { orderCreator } from '../../lib/orders';
import { productionDeviations, productionStatus, sheetStyle } from '../../lib/production';

// BakerOrderCard is a compact matrix cell card: tap opens the quick-look
// popup (or toggles the checkbox in selection mode). The card carries the
// position count and the author — shop, date and тип are given by the matrix
// axes and the stripe color; the number lives in the popup.
//
// Язык состояний (без бейджей — они громоздки на узких карточках):
// - отработан: карточка целиком в цвете партии + «✓» (или «±N» при
//   отклонениях) в углу; заказы одного листа читаются как цветовая группа;
// - отменён: «погашенная» карточка — блёклая, счётчик зачёркнут, «✕» в углу.
export function BakerOrderCard({ order, selected, checked, selectionMode, onToggleSelection, onPreview }) {
  const handleClick = selectionMode ? onToggleSelection : onPreview;
  const style = order.category ? categoryStyle(order.category) : null;
  const production = order.cancelled ? null : productionStatus(order);
  const sheet = production ? sheetStyle(order.production_sheet_id) : null;
  const deviations = production ? productionDeviations(order) : 0;

  const surface = checked
    ? 'border-stone-900 bg-white ring-1 ring-stone-900/15'
    : selected && !selectionMode
      ? 'border-stone-800 bg-white shadow-sm'
      : order.cancelled
        ? 'border-stone-200 bg-stone-100/70 hover:border-stone-400'
        : sheet
          ? `${sheet.card} shadow-sm`
          : 'border-stone-200 bg-white hover:border-stone-400 hover:shadow-sm';

  return (
    <button
      type="button"
      onClick={handleClick}
      data-testid="bakerOrderCard-root"
      data-order-number={order.number}
      title={production ? `${production.label} · лист №${order.production_sheet_id}` : order.cancelled ? 'Заказ отменён' : undefined}
      aria-label={`${order.items?.length || 0} позиций, ${orderCreator(order)}${production ? `, ${production.label}, лист номер ${order.production_sheet_id}` : ''}${order.cancelled ? ', отменён' : ''}`}
      className={`group relative w-full rounded-lg border px-2 py-1.5 text-left transition duration-150 focus:outline-none focus:ring-2 focus:ring-stone-900/20 active:scale-[0.98] ${
        style ? `border-l-4 ${style.stripe}` : ''
      } ${surface}`}
    >
      <span className="flex items-baseline justify-between gap-1">
        <strong
          className={`min-w-0 truncate text-body font-semibold ${
            order.cancelled ? 'text-stone-500 line-through decoration-stone-400' : 'text-stone-950'
          }`}
        >
          {order.items?.length || 0} поз.
        </strong>
        {order.cancelled ? (
          <span className="shrink-0 text-body font-bold leading-5 text-red-500" aria-hidden="true">✕</span>
        ) : production ? (
          <span className={`shrink-0 text-body font-bold leading-5 tabular-nums ${sheet.check}`} aria-hidden="true">
            {deviations > 0 ? `±${deviations}` : '✓'}
          </span>
        ) : null}
      </span>
      <span className={`mt-0.5 block truncate text-caption ${order.cancelled ? 'text-stone-400' : production ? 'text-stone-600' : 'text-stone-500'}`}>
        {orderCreator(order)}
      </span>
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
