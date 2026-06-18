import { Icon } from '../../components/Icon';
import { formatFulfillmentDate } from '../../lib/format';
import { orderSource } from '../../lib/orders';

export function BakerOrderCard({ order, selected, checked, selectionMode, onSelect, onToggleSelection, onPreview }) {
  const handleClick = selectionMode ? onToggleSelection : onSelect;

  return (
    <article
      className={`group relative flex flex-col rounded-xl border bg-white p-3 transition ${
        checked
          ? 'border-stone-900 ring-1 ring-stone-900/15'
          : selected
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
          className={`absolute right-2.5 top-2.5 inline-flex h-6 w-6 items-center justify-center rounded-full border text-[11px] font-semibold transition ${
            checked ? 'border-stone-900 bg-stone-900 text-white' : 'border-stone-300 bg-white text-transparent group-hover:border-stone-400'
          }`}
          aria-hidden="true"
        >
          <Icon name="select" size={13} />
        </span>
      ) : (
        <button
          type="button"
          onClick={onPreview}
          title="Обзор"
          aria-label="Обзор заказа"
          className="absolute right-2 top-2 inline-flex h-7 w-7 items-center justify-center rounded-md text-stone-400 opacity-0 transition hover:bg-stone-100 hover:text-stone-700 focus:opacity-100 focus:outline-none focus:ring-2 focus:ring-stone-900/20 group-hover:opacity-100 sm:opacity-100"
        >
          <Icon name="eye" size={16} />
        </button>
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
