import { Button } from '../../components/Button';
import { EmptyState } from '../../components/EmptyState';
import { formatFulfillmentDate } from '../../lib/format';
import { orderSource } from '../../lib/orders';

export function OrderList({ loading, orders, page, selectedNumber, onRefresh, onSelect, onPageChange }) {
  return (
    <aside className="m-3 flex max-h-[44vh] flex-col gap-2.5 rounded-lg border border-stone-200 bg-white p-3 lg:sticky lg:top-3 lg:ml-3 lg:mr-0 lg:h-[calc(100vh-1.5rem)] lg:max-h-none">
      <div className="flex items-center justify-between gap-2">
        <div className="min-w-0">
          <h1 className="m-0 text-[20px] font-semibold leading-7 text-stone-950">Заказы</h1>
        </div>
      </div>

      <Button variant="primary" className="self-start" onClick={onRefresh} disabled={loading}>
        Обновить
      </Button>

      <div className="min-h-0 flex-1 space-y-1 overflow-y-auto pr-1">
        {orders.length ? (
          orders.map((order) => (
            <button
              key={order.number}
              className={`w-full rounded-md border px-2.5 py-2 text-left transition ${
                selectedNumber === order.number ? 'border-stone-300 bg-stone-100' : 'border-transparent bg-white hover:border-stone-300 hover:bg-stone-50'
              }`}
              onClick={() => onSelect(order.number)}
            >
              <div className="grid grid-cols-[minmax(0,1fr)_auto] gap-x-3 gap-y-0.5">
                <strong className="break-words text-[13px] font-semibold leading-5 text-stone-900">{order.number}</strong>
                <span className="text-[12px] leading-5 text-stone-500">{order.items?.length || 0} поз.</span>
                <strong className="break-words text-[12px] font-semibold leading-5 text-stone-800">Откуда: {orderSource(order)}</strong>
                <span className="text-[12px] leading-5 text-stone-500">{formatFulfillmentDate(order.fulfillment_date) || '-'}</span>
              </div>
            </button>
          ))
        ) : (
          <EmptyState compact>Заказов нет.</EmptyState>
        )}
      </div>

      <div className="grid grid-cols-[1fr_auto_1fr] items-center gap-2">
        <Button onClick={() => onPageChange(page.page - 1)} disabled={loading || page.page <= 1}>
          Назад
        </Button>
        <span className="whitespace-nowrap text-xs text-slate-500">
          {page.page} / {page.total_pages || 1}
        </span>
        <Button onClick={() => onPageChange(page.page + 1)} disabled={loading || page.page >= (page.total_pages || 1)}>
          Далее
        </Button>
      </div>
    </aside>
  );
}
