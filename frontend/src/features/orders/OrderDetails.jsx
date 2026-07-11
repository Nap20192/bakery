import { CategoryBadge } from '../../ui/CategoryBadge';
import { MetaCell } from '../../ui/MetaCell';
import { PanelHeader } from '../../ui/Panel';
import { CopyButton } from '../../ui/CopyButton';
import { Button } from '../../ui/Button';
import { Icon } from '../../ui/Icon';
import { formatDate, formatFulfillmentDate, formatQuantity } from '../../lib/format';
import { cn } from '../../lib/cn';
import { orderCreator, orderItemsToText, orderQuantity } from '../../lib/orders';
import { productionStatus, sheetStyle } from '../../lib/production';
import { EmptyState } from '../../ui/EmptyState';

export function OrderDetails({ order, catalog = [], canFavorite = false, onToggleFavorite, onOpenProduction }) {
  const production = productionStatus(order);
  return (
    <>
      <div className="flex items-start justify-between gap-2">
        <div className="flex min-w-0 flex-wrap items-center gap-2">
          <PanelHeader title={order.number} count={order.items?.length || 0} />
          <CategoryBadge category={order.category} />
          {order.cancelled && (
            <span className="inline-flex items-center rounded-full border border-red-300 bg-red-50 px-2 py-0.5 text-note font-semibold text-red-700">
              Отменён
            </span>
          )}
        </div>
        <div className="flex shrink-0 items-center gap-1.5 sm:gap-2">
          {order.production_sheet_id && onOpenProduction && (
            // Кнопка перехода в цвете партии — тот же цвет, что у бейджа
            // этого листа на карточках матрицы и в журнале.
            <Button
              onClick={() => onOpenProduction(order.production_sheet_id)}
              title="Открыть отработку в журнале"
              className={cn(sheetStyle(order.production_sheet_id).badge, 'hover:opacity-85')}
            >
              <Icon name="calculator" size={15} />
              <span className="hidden sm:inline">{production?.deviations ? `±${production.deviations} · Отработка №${order.production_sheet_id}` : `✓ Отработан · №${order.production_sheet_id}`}</span>
              <span className="sm:hidden">№{order.production_sheet_id}</span>
            </Button>
          )}
          {canFavorite ? (
            <Button
              variant={order.favorite ? 'primary' : 'default'}
              onClick={() => onToggleFavorite?.(order.number, !order.favorite)}
              title={order.favorite ? 'В избранном — не удаляется' : 'В избранное'}
              aria-label={order.favorite ? 'В избранном' : 'В избранное'}
            >
              <Icon name="star" size={15} filled={Boolean(order.favorite)} />
              <span className="hidden sm:inline">{order.favorite ? 'В избранном' : 'В избранное'}</span>
            </Button>
          ) : (
            order.favorite && (
              <span className="inline-flex items-center gap-1 text-note font-medium text-stone-500" title="Избранный заказ">
                <Icon name="star" size={14} filled />
                <span className="hidden sm:inline">Избранное</span>
              </span>
            )
          )}
          {(order.items?.length || 0) > 0 && (
            <CopyButton getText={() => orderItemsToText(order, false)} label="Копировать" labelClassName="hidden sm:inline" />
          )}
        </div>
      </div>

      {production && (
        <div className="mb-3 flex items-start gap-2 rounded-lg border border-stone-300 bg-stone-100/80 px-3 py-2 text-body text-stone-800" role="status">
          <span className="mt-0.5 font-bold" aria-hidden="true">{production.deviations > 0 ? '±' : '✓'}</span>
          <div>
            <strong className="block font-semibold">{production.label}</strong>
            <span className="block text-note text-stone-600">Заказ входит в отработку №{order.production_sheet_id}; расчёт теста использует зафиксированный факт.</span>
          </div>
        </div>
      )}

      <div className="mb-3 grid grid-cols-2 gap-1.5 sm:gap-2 md:grid-cols-5">
        <MetaCell label="Создан" value={formatDate(order.created_at) || '-'} />
        <MetaCell label="Выполнить" value={formatFulfillmentDate(order.fulfillment_date) || '-'} />
        <MetaCell label="От кого" value={orderCreator(order)} />
        <MetaCell label="Откуда" value={order.from_department?.name || order.location || '-'} />
        <MetaCell label="Куда" value={order.to_department?.name || '-'} />
      </div>

      <OrderItems items={order.items || []} history={order.history || []} comments={commentByName(order)} catalog={catalog} />
      {order.comments?.general && (
        <div className="mt-3 rounded-lg border border-stone-200 bg-stone-50 px-3 py-2.5">
          <span className="block text-caption font-medium uppercase leading-4 text-stone-500">Комментарий</span>
          <p className="m-0 mt-0.5 whitespace-pre-wrap break-words text-body leading-5 text-stone-800">{order.comments.general}</p>
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

function OrderItems({ items, history, comments = {}, catalog = [] }) {
  if (!items.length) {
    return <EmptyState compact>В заказе нет позиций.</EmptyState>;
  }
  const latestChanges = latestHistoryByCode(history);
  const totalQuantity = items.reduce((sum, item) => sum + (Number(item.quantity) || 0) + (Number(item.reserved_quantity) || 0), 0);
  // Колонка «Испечено» появляется только после внесения отработки.
  const hasProduction = items.some((item) => item.produced_quantity != null);
  const totalProduced = items.reduce((sum, item) => sum + (Number(item.produced_quantity) || 0), 0);
  const groups = groupItemsByTheme(items, catalog);
  const showHeaders = groups.length > 1;
  const rowGrid = hasProduction
    ? 'grid-cols-[minmax(0,1fr)_4.2rem_4.2rem] sm:grid-cols-[minmax(0,1fr)_5rem_5rem]'
    : 'grid-cols-[minmax(0,1fr)_4.6rem] sm:grid-cols-[minmax(0,1fr)_5rem]';
  return (
    <div className="overflow-hidden rounded-lg border border-stone-300 bg-white">
      <div className={`grid ${rowGrid} gap-2 bg-stone-50 px-2.5 py-1.5 text-caption font-medium uppercase leading-5 text-stone-600 sm:px-3`}>
        <span>Позиция</span>
        <span className="text-right">Кол-во</span>
        {hasProduction && <span className="text-right">Испечено</span>}
      </div>
      {groups.map((group) => (
        <div key={group.theme}>
          {showHeaders && (
            <div className="border-t border-stone-300 bg-stone-50/70 px-2.5 py-1 text-caption font-semibold uppercase tracking-wide text-stone-500 sm:px-3">
              {group.theme}
            </div>
          )}
          <div className="divide-y divide-stone-200 border-t border-stone-300">
            {group.items.map((item) => {
              const change = latestChanges[item.code];
              return (
                <div
                  className={`grid ${rowGrid} items-start gap-2 border-l-4 px-2.5 py-1.5 text-body sm:px-3 ${
                    change ? changeStyle(change.change_type).row : 'border-l-transparent'
                  }`}
                  key={item.code || item.product_name}
                >
                  <span className="min-w-0 break-words leading-5 text-stone-800">
                    {item.product_name}
                    {change && (
                      <span className={`ml-1 whitespace-nowrap text-caption font-semibold ${changeStyle(change.change_type).text}`}>
                        {changeStyle(change.change_type).label}
                      </span>
                    )}
                    {comments[item.product_name] && (
                      <span className="mt-0.5 block break-words text-note leading-4 text-stone-500">{comments[item.product_name]}</span>
                    )}
                    {item.produced_reason && (
                      <span className="mt-0.5 block break-words text-note leading-4 text-sky-700">Отработка: {item.produced_reason}</span>
                    )}
                  </span>
                  <span className="text-right text-note font-semibold leading-5 text-stone-950 sm:text-body">{orderQuantity(item)}</span>
                  {hasProduction && <ProducedCell item={item} />}
                </div>
              );
            })}
          </div>
        </div>
      ))}
      <div className={`grid ${rowGrid} gap-2 border-t border-stone-300 bg-stone-50 px-2.5 py-1.5 text-note font-semibold leading-5 text-stone-900 sm:px-3`}>
        <span>Итого</span>
        <span className="text-right tabular-nums">{formatQuantity(totalQuantity)}</span>
        {hasProduction && <span className="text-right tabular-nums">{formatQuantity(totalProduced)}</span>}
      </div>
    </div>
  );
}

// ProducedCell показывает факт выпечки: недовоз красным, излишек зелёным.
function ProducedCell({ item }) {
  if (item.produced_quantity == null) {
    return <span className="text-right text-note leading-5 text-stone-300">—</span>;
  }
  const ordered = Number(item.production_quantity ?? (Number(item.quantity) || 0) + (Number(item.reserved_quantity) || 0));
  const produced = Number(item.produced_quantity);
  const tone = produced < ordered ? 'text-red-700' : produced > ordered ? 'text-emerald-700' : 'text-stone-700';
  return (
    <span className={`text-right text-note font-semibold leading-5 tabular-nums sm:text-body ${tone}`}>
      {formatQuantity(produced)}
    </span>
  );
}

// groupItemsByTheme splits order items into catalog theme groups, keeping the
// catalog's group order. Items not found in the catalog fall into «Прочее».
function groupItemsByTheme(items, catalog) {
  const themeByCode = new Map();
  const themeByName = new Map();
  const themeOrder = [];
  for (const dish of catalog) {
    const theme = (dish.theme || '').trim() || 'Без группы';
    if (dish.code) themeByCode.set(dish.code, theme);
    themeByName.set(String(dish.name || '').toLowerCase().trim(), theme);
    if (!themeOrder.includes(theme)) themeOrder.push(theme);
  }
  const byTheme = new Map();
  for (const item of items) {
    let theme = themeByCode.get(item.code);
    if (theme === undefined) theme = themeByName.get(String(item.product_name || '').toLowerCase().trim());
    if (theme === undefined) theme = 'Прочее';
    if (!byTheme.has(theme)) byTheme.set(theme, { theme, items: [] });
    byTheme.get(theme).items.push(item);
  }
  const groups = [];
  for (const theme of themeOrder) {
    if (byTheme.has(theme)) {
      groups.push(byTheme.get(theme));
      byTheme.delete(theme);
    }
  }
  for (const [theme, group] of byTheme) {
    if (theme !== 'Прочее') groups.push(group);
  }
  if (byTheme.has('Прочее')) groups.push(byTheme.get('Прочее'));
  return groups;
}

function OrderHistory({ history }) {
  if (!history.length) return null;
  return (
    <section className="mt-3 overflow-hidden rounded-lg border border-stone-300 bg-white">
      <h4 className="border-b border-stone-300 bg-stone-50 px-2.5 py-2 text-body font-semibold leading-5 text-stone-950 sm:px-3">История изменений</h4>
      <div className="divide-y divide-stone-300">
        {history.map((entry) => (
          <article className="px-2.5 py-2 sm:px-3" key={entry.id}>
            <div className="mb-1.5 flex flex-wrap items-center gap-x-2 gap-y-1 text-note leading-5 text-stone-600">
              <strong className="text-stone-800">{formatDate(entry.changed_at) || '-'}</strong>
              {entry.changed_by_username && <span>@{entry.changed_by_username.replace(/^@/, '')}</span>}
            </div>
            <div className="space-y-1">
              {(entry.items || []).map((item) => (
                <div className="text-body leading-5" key={`${entry.id}-${item.product_code}-${item.change_type}`}>
                  <span className="min-w-0 break-words text-stone-800">
                    <span className={`font-semibold ${changeStyle(item.change_type).text}`}>
                      {changeStyle(item.change_type).label}
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

// Стили типов изменений — Tailwind-классы вместо инлайновых hex: строка
// позиции получает кромку и подложку, метка в тексте — цвет. Семантика
// данных (зелёный/янтарный/красный), токенами не тонируется.
const CHANGE_STYLES = {
  added: { label: '[добавлено]', row: 'border-l-emerald-500 bg-emerald-50/70', text: 'text-emerald-700' },
  updated: { label: '[изменено]', row: 'border-l-amber-400 bg-amber-50/70', text: 'text-amber-600' },
  removed: { label: '[удалено]', row: 'border-l-red-400 bg-red-50/70', text: 'text-red-600' },
};
const CHANGE_FALLBACK = { label: '[обновлено]', row: 'border-l-stone-300 bg-stone-50', text: 'text-stone-600' };

function changeStyle(type) {
  return CHANGE_STYLES[type] || CHANGE_FALLBACK;
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
  if (item.change_type === 'produced') {
    // Для отработки old/new хранят прежний и новый факт; new может быть пуст
    // при снятии отработки.
    if (item.new_quantity == null) return `факт ${formatQuantity(item.old_quantity || 0)} снят`;
    if (item.old_quantity == null) return `факт ${formatQuantity(item.new_quantity)}`;
    return `факт ${formatQuantity(item.old_quantity)} -> ${formatQuantity(item.new_quantity)}`;
  }
  return '';
}

function historyQuantity(quantity, reserved) {
  const base = formatQuantity(quantity || 0);
  if (!reserved) return base;
  return `${base}+${formatQuantity(reserved)}`;
}
