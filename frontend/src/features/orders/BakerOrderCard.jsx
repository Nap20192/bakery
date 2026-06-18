import { Button } from '../../components/Button';
import { Icon } from '../../components/Icon';
import { formatFulfillmentDate } from '../../lib/format';
import { orderSource } from '../../lib/orders';

export function BakerOrderCard({ order, selected, checked, selectionMode, onSelect, onToggleSelection, onPreview }) {
  // Plain tap opens the quick-look popup; the explicit button opens the full
  // calculation screen. In selection mode a tap toggles the checkbox.
  const handleClick = selectionMode ? onToggleSelection : onPreview;

  return (
    <article
      className={`group relative flex flex-col rounded-xl border bg-white p-3 transition duration-150 active:scale-[0.99] ${
        checked
          ? 'border-stone-900 ring-1 ring-stone-900/15'
          : !selectionMode && selected
            ? 'border-stone-800 shadow-sm'
            : 'border-stone-200 hover:border-stone-400 hover:shadow-sm'
      }`}
    >
      <button
        type="button"
        onClick={handleClick}
        className="min-w-0 rounded-md text-left focus:outline-none focus:ring-2 focus:ring-stone-900/20"
      >
        <div className="flex items-center gap-2">
          <strong className="min-w-0 flex-1 truncate text-[15px] font-semibold leading-6 text-stone-950">{order.number}</strong>
          <span className="shrink-0 rounded-full bg-stone-100 px-2 py-0.5 text-[11px] font-medium tabular-nums text-stone-600">
            {order.items?.length || 0}
          </span>
        </div>
        <span className="mt-0.5 block truncate text-[12px] leading-5 text-stone-500">{orderSource(order)}</span>
      </button>

      <button
        type="button"
        onClick={handleClick}
        className="mt-2.5 space-y-1 rounded-md text-left focus:outline-none focus:ring-2 focus:ring-stone-900/20"
      >
        <Fact label="Выполнить" value={formatFulfillmentDate(order.fulfillment_date) || '—'} strong />
        <Fact label="Куда" value={order.to_department?.name || '—'} />
      </button>

      {selectionMode ? (
        <span
          className={`absolute right-2.5 top-2.5 inline-flex h-6 w-6 items-center justify-center rounded-full border transition duration-150 ${
            checked ? 'border-stone-900 bg-stone-900 text-white' : 'border-stone-300 bg-white text-transparent group-hover:border-stone-400'
          }`}
          aria-hidden="true"
        >
          <Icon name="select" size={13} />
        </span>
      ) : (
        <div className="mt-2.5 border-t border-stone-100 pt-2.5">
          <Button onClick={onSelect} className="w-full">
            <Icon name="calculator" size={15} />
            Расчёт теста
          </Button>
        </div>
      )}
    </article>
  );
}

function Fact({ label, value, strong = false }) {
  return (
    <span className="flex items-baseline justify-between gap-2 text-[12px] leading-5">
      <span className="shrink-0 text-stone-400">{label}</span>
      <span className={`min-w-0 truncate text-right text-stone-800 ${strong ? 'font-semibold text-stone-950' : 'font-medium'}`}>{value}</span>
    </span>
  );
}
