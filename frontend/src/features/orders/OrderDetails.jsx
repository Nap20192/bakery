import { MetaCell } from '../../components/MetaCell';
import { PanelHeader } from '../../components/Panel';
import { formatDate, formatFulfillmentDate } from '../../lib/format';
import { orderCreator, orderQuantity } from '../../lib/orders';
import { EmptyState } from '../../components/EmptyState';

export function OrderDetails({ order }) {
  return (
    <>
      <PanelHeader title={order.number} count={order.items?.length || 0} />

      <div className="mb-3 grid grid-cols-2 gap-1.5 sm:gap-2 md:grid-cols-5">
        <MetaCell label="Создан" value={formatDate(order.created_at) || '-'} />
        <MetaCell label="Выполнить" value={formatFulfillmentDate(order.fulfillment_date) || '-'} />
        <MetaCell label="От кого" value={orderCreator(order)} />
        <MetaCell label="Откуда" value={order.from_department?.name || order.location || '-'} />
        <MetaCell label="Куда" value={order.to_department?.name || '-'} />
      </div>

      <OrderItems items={order.items || []} />
    </>
  );
}

function OrderItems({ items }) {
  if (!items.length) {
    return <EmptyState compact>В заказе нет позиций.</EmptyState>;
  }
  return (
    <div className="overflow-hidden rounded-lg border border-stone-200 bg-white">
      <div className="grid grid-cols-[4.2rem_minmax(0,1fr)_4.6rem] gap-2 bg-stone-50 px-2.5 py-1.5 text-[11px] font-medium uppercase leading-5 text-stone-500 sm:grid-cols-[5rem_minmax(0,1fr)_5rem] sm:px-3">
        <span>Код</span>
        <span>Позиция</span>
        <span className="text-right">Кол-во</span>
      </div>
      <div className="divide-y divide-stone-200">
        {items.map((item) => (
          <div
            className="grid grid-cols-[4.2rem_minmax(0,1fr)_4.6rem] items-start gap-2 px-2.5 py-1.5 text-[13px] sm:grid-cols-[5rem_minmax(0,1fr)_5rem] sm:px-3"
            key={item.code}
          >
            <code className="break-words text-[12px] leading-5 text-stone-500">{item.code}</code>
            <span className="min-w-0 break-words leading-5 text-stone-800">{item.product_name}</span>
            <span className="text-right text-[12px] font-semibold leading-5 text-stone-950 sm:text-[13px]">{orderQuantity(item)}</span>
          </div>
        ))}
      </div>
    </div>
  );
}
