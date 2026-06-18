import { MetaCell } from '../../components/MetaCell';
import { PanelHeader } from '../../components/Panel';
import { CopyButton } from '../../components/CopyButton';
import { formatDate, formatFulfillmentDate, formatQuantity } from '../../lib/format';
import { orderCreator, orderItemsToText, orderQuantity } from '../../lib/orders';
import { EmptyState } from '../../components/EmptyState';

export function OrderDetails({ order }) {
  return (
    <>
      <div className="flex items-start justify-between gap-2">
        <PanelHeader title={order.number} count={order.items?.length || 0} />
        {(order.items?.length || 0) > 0 && (
          <CopyButton getText={() => orderItemsToText(order, false)} label="Копировать" className="shrink-0" />
        )}
      </div>

      <div className="mb-3 grid grid-cols-2 gap-1.5 sm:gap-2 md:grid-cols-5">
        <MetaCell label="Создан" value={formatDate(order.created_at) || '-'} />
        <MetaCell label="Выполнить" value={formatFulfillmentDate(order.fulfillment_date) || '-'} />
        <MetaCell label="От кого" value={orderCreator(order)} />
        <MetaCell label="Откуда" value={order.from_department?.name || order.location || '-'} />
        <MetaCell label="Куда" value={order.to_department?.name || '-'} />
      </div>

      <OrderItems items={order.items || []} history={order.history || []} comments={commentByName(order)} />
      {order.comments?.general && (
        <div className="mt-3 rounded-lg border border-stone-200 bg-stone-50 px-3 py-2.5">
          <span className="block text-[11px] font-medium uppercase leading-4 text-stone-500">Комментарий</span>
          <p className="m-0 mt-0.5 whitespace-pre-wrap break-words text-[13px] leading-5 text-stone-800">{order.comments.general}</p>
        </div>
      )}
      <OrderHistory history={order.history || []} />
    </>
  );
}

function commentByName(order) {
  const map = {};
  for (const entry of order?.comments?.items || []) {
    if (entry.product_name && entry.comment) map[entry.product_name] = entry.comment;
  }
  return map;
}

function OrderItems({ items, history, comments = {} }) {
  if (!items.length) {
    return <EmptyState compact>В заказе нет позиций.</EmptyState>;
  }
  const latestChanges = latestHistoryByCode(history);
  const totalQuantity = items.reduce((sum, item) => sum + (Number(item.quantity) || 0) + (Number(item.reserved_quantity) || 0), 0);
  return (
    <div className="overflow-hidden rounded-lg border border-stone-300 bg-white">
      <div className="grid grid-cols-[minmax(0,1fr)_4.6rem] gap-2 bg-stone-50 px-2.5 py-1.5 text-[11px] font-medium uppercase leading-5 text-stone-600 sm:grid-cols-[minmax(0,1fr)_5rem] sm:px-3">
        <span>Позиция</span>
        <span className="text-right">Кол-во</span>
      </div>
      <div className="divide-y divide-stone-300">
        {items.map((item) => {
          const change = latestChanges[item.code];
          return (
            <div
              className="grid grid-cols-[minmax(0,1fr)_4.6rem] items-start gap-2 border-l-4 px-2.5 py-1.5 text-[13px] sm:grid-cols-[minmax(0,1fr)_5rem] sm:px-3"
              style={{
                borderLeftColor: change ? changeColor(change.change_type) : 'transparent',
                backgroundColor: change ? changeBackground(change.change_type) : 'transparent',
              }}
              key={item.code}
            >
              <span className="min-w-0 break-words leading-5 text-stone-800">
                {item.product_name}
                {change && (
                  <span className="ml-1 whitespace-nowrap text-[11px] font-semibold" style={{ color: changeColor(change.change_type) }}>
                    {changeLabel(change.change_type)}
                  </span>
                )}
                {comments[item.product_name] && (
                  <span className="mt-0.5 block break-words text-[12px] leading-4 text-stone-500">{comments[item.product_name]}</span>
                )}
              </span>
              <span className="text-right text-[12px] font-semibold leading-5 text-stone-950 sm:text-[13px]">{orderQuantity(item)}</span>
            </div>
          );
        })}
      </div>
      <div className="grid grid-cols-[minmax(0,1fr)_4.6rem] gap-2 border-t border-stone-300 bg-stone-50 px-2.5 py-1.5 text-[12px] font-semibold leading-5 text-stone-900 sm:grid-cols-[minmax(0,1fr)_5rem] sm:px-3">
        <span>Итого</span>
        <span className="text-right tabular-nums">{formatQuantity(totalQuantity)}</span>
      </div>
    </div>
  );
}

function OrderHistory({ history }) {
  if (!history.length) return null;
  return (
    <section className="mt-3 overflow-hidden rounded-lg border border-stone-300 bg-white">
      <h4 className="border-b border-stone-300 bg-stone-50 px-2.5 py-2 text-[13px] font-semibold leading-5 text-stone-950 sm:px-3">История изменений</h4>
      <div className="divide-y divide-stone-300">
        {history.map((entry) => (
          <article className="px-2.5 py-2 sm:px-3" key={entry.id}>
            <div className="mb-1.5 flex flex-wrap items-center gap-x-2 gap-y-1 text-[12px] leading-5 text-stone-600">
              <strong className="text-stone-800">{formatDate(entry.changed_at) || '-'}</strong>
              {entry.changed_by_username && <span>@{entry.changed_by_username.replace(/^@/, '')}</span>}
            </div>
            <div className="space-y-1">
              {(entry.items || []).map((item) => (
                <div className="text-[13px] leading-5" key={`${entry.id}-${item.product_code}-${item.change_type}`}>
                  <span className="min-w-0 break-words text-stone-800">
                    <span className="font-semibold" style={{ color: changeColor(item.change_type) }}>
                      {changeLabel(item.change_type)}
                    </span>{' '}
                    {item.product_name}
                    <span className="text-stone-600"> {historyQuantityText(item)}</span>
                  </span>
                </div>
              ))}
            </div>
          </article>
        ))}
      </div>
    </section>
  );
}

function latestHistoryByCode(history) {
  const result = {};
  const latest = history?.[0];
  for (const item of latest?.items || []) {
    if (item.product_code) result[item.product_code] = item;
  }
  return result;
}

function changeLabel(type) {
  if (type === 'added') return '[добавлено]';
  if (type === 'updated') return '[изменено]';
  if (type === 'removed') return '[удалено]';
  return '[обновлено]';
}

function changeColor(type) {
  if (type === 'added') return '#1f9d55';
  if (type === 'updated') return '#ffaf00';
  if (type === 'removed') return '#d64545';
  return '#57534e';
}

function changeBackground(type) {
  if (type === 'added') return '#e7f6ed';
  if (type === 'updated') return '#fff0c2';
  if (type === 'removed') return '#fde8e8';
  return '#f5f5f4';
}

function historyQuantityText(item) {
  if (item.change_type === 'updated') {
    return `${historyQuantity(item.old_quantity, item.old_reserved_quantity)} -> ${historyQuantity(item.new_quantity, item.new_reserved_quantity)}`;
  }
  if (item.change_type === 'added') {
    return historyQuantity(item.new_quantity, item.new_reserved_quantity);
  }
  if (item.change_type === 'removed') {
    return `было ${historyQuantity(item.old_quantity, item.old_reserved_quantity)}`;
  }
  return '';
}

function historyQuantity(quantity, reserved) {
  const base = formatQuantity(quantity || 0);
  if (!reserved) return base;
  return `${base}+${formatQuantity(reserved)}`;
}
