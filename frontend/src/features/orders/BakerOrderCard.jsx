import { Button } from '../../components/Button';
import { Icon } from '../../components/Icon';
import { formatDate, formatFulfillmentDate } from '../../lib/format';
import { orderSource } from '../../lib/orders';

export function BakerOrderCard({ order, selected, checked, selectionMode, onSelect, onToggleSelection, onPreview }) {
  const handleClick = selectionMode ? onToggleSelection : onSelect;

  return (
    <article
      className={`rounded-lg border bg-white p-2.5 shadow-sm transition hover:shadow-md ${
        checked
          ? 'border-stone-950 ring-2 ring-stone-900/10'
          : selected
            ? 'border-stone-800'
            : 'border-stone-300 hover:border-stone-500'
      }`}
    >
      <div className="mb-2 flex items-start justify-between gap-2">
        <button type="button" onClick={handleClick} className="min-w-0 flex-1 rounded-md text-left focus:outline-none focus:ring-2 focus:ring-stone-900/20">
          <strong className="block truncate text-[15px] font-semibold leading-6 text-stone-950">{order.number}</strong>
          <span className="block truncate text-[12px] leading-5 text-stone-600">{orderSource(order)}</span>
        </button>
        {selectionMode && (
          <span className={`shrink-0 rounded-md border px-2 py-1 text-[11px] font-semibold ${checked ? 'border-stone-950 bg-stone-950 text-white' : 'border-stone-300 text-stone-600'}`}>
            {checked && <Icon name="select" size={13} className="mr-1 inline-block align-[-2px]" />}
            {checked ? 'Выбран' : 'Выбрать'}
          </span>
        )}
      </div>

      <button type="button" onClick={handleClick} className="grid w-full grid-cols-2 gap-1.5 rounded-md text-left focus:outline-none focus:ring-2 focus:ring-stone-900/20">
        <CardMeta label="Выполнить" value={formatFulfillmentDate(order.fulfillment_date) || '-'} strong />
        <CardMeta label="Позиций" value={String(order.items?.length || 0)} />
        <CardMeta label="Создан" value={formatDate(order.created_at) || '-'} />
        <CardMeta label="Куда" value={order.to_department?.name || '-'} />
      </button>

      <div className="mt-2 flex justify-end">
        <Button onClick={onPreview}>
          <Icon name="eye" size={16} />
          Обзор
        </Button>
      </div>
    </article>
  );
}

function CardMeta({ label, value, strong = false }) {
  return (
    <span className="min-w-0 rounded-md border border-stone-200 bg-stone-50 px-2 py-1">
      <span className="block text-[10px] font-medium uppercase leading-4 text-stone-500">{label}</span>
      <span className={`block truncate text-[12px] leading-5 text-stone-900 ${strong ? 'font-semibold' : 'font-medium'}`}>{value}</span>
    </span>
  );
}
