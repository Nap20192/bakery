import { useState } from 'react';
import { Button } from '../../ui/Button';
import { CategoryBadge } from '../../ui/CategoryBadge';
import { ErrorBanner } from '../../ui/ErrorBanner';
import { EmptyState } from '../../ui/EmptyState';
import { Icon } from '../../ui/Icon';
import { panelClass, PanelHeader } from '../../ui/Panel';
import { MonitorReports } from './MonitorReports';
import { OrderDetails } from '../orders/OrderDetails';
import { ProductionSheet } from '../production/ProductionSheet';
import { plural } from '../../lib/format';
import { orderCreator } from '../../lib/orders';
import { productionDeviations, productionStatus, sheetStyle } from '../../lib/production';

// CollapsedOrder — заказ партии одной строкой: раскрывается по клику до
// полных деталей. Свёрнутые заказы не мешают работать с отработкой и
// расчётом — страница делится на понятные разделы.
function CollapsedOrder({ order, catalog, expanded, onToggle, onRemove, onOpenProduction, loading }) {
  const production = order.cancelled ? null : productionStatus(order);
  const sheet = production ? sheetStyle(order.production_sheet_id) : null;
  const deviations = production ? productionDeviations(order) : 0;
  return (
    <div className="py-1">
      <div className="flex items-center gap-1.5">
        <button
          type="button"
          onClick={onToggle}
          aria-expanded={expanded}
          className="flex min-h-11 min-w-0 flex-1 items-center gap-2 rounded-md px-1.5 text-left transition hover:bg-stone-50 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring/30"
        >
          <Icon name="chevronRight" size={14} className={`shrink-0 text-stone-400 transition-transform ${expanded ? 'rotate-90' : ''}`} />
          <span className="min-w-0 truncate text-body font-semibold text-stone-950">{order.number}</span>
          <CategoryBadge category={order.category} />
          <span className="shrink-0 text-note tabular-nums text-stone-500">{order.items?.length || 0} поз.</span>
          <span className="hidden min-w-0 truncate text-note text-stone-500 sm:inline">{orderCreator(order)}</span>
          {production && (
            <span className={`ml-auto shrink-0 text-body font-bold tabular-nums ${sheet.check}`} aria-hidden="true">
              {deviations > 0 ? `±${deviations}` : '✓'}
            </span>
          )}
        </button>
        <Button
          variant="ghost"
          onClick={onRemove}
          disabled={loading}
          aria-label={`Убрать ${order.number} из выбора`}
          className="shrink-0 px-2"
        >
          <Icon name="close" size={15} />
        </Button>
      </div>
      {expanded && (
        <div className="mt-1 border-t border-stone-100 pt-3">
          <OrderDetails order={order} catalog={catalog} onOpenProduction={onOpenProduction} />
        </div>
      )}
    </div>
  );
}

export function BakerSelectionReview({
  loading,
  selectedOrders,
  catalog = [],
  monitor,
  error,
  onBack,
  onRemove,
  onCalculate,
  onSaveProduction,
  onOpenJournal,
  onOpenProduction,
}) {
  const [expanded, setExpanded] = useState(() => new Set());
  // Констрейнт (docs/constraints.md): расчёт теста доступен только при
  // сохранённой отработке. Несохранённые правки листа блокируют расчёт и
  // прячут устаревший результат; ключ перемонтирует редактор после
  // сохранения (меняется сохранённый факт заказов).
  const [productionDirty, setProductionDirty] = useState(false);
  const sheetKey = selectedOrders
    .map((order) => `${order.number}:${order.production_sheet_id ?? ''}:${(order.items || []).map((item) => item.produced_quantity ?? '').join(',')}`)
    .join('|');

  function toggle(number) {
    setExpanded((current) => {
      const next = new Set(current);
      if (next.has(number)) next.delete(number);
      else next.add(number);
      return next;
    });
  }

  return (
    <section className="px-3 py-3 pb-20 sm:px-5 sm:pb-3 lg:px-6">
      <div className="mx-auto max-w-[1440px] space-y-4">
        <section className="rounded-lg border border-stone-300 bg-white p-3 shadow-sm">
          <div className="flex flex-wrap items-center justify-between gap-2">
            <div>
              <h1 className="m-0 text-page font-semibold text-stone-950">Выбранные заказы</h1>
              <p className="m-0 text-body text-stone-600">
                {selectedOrders.length} {plural(selectedOrders.length, 'заказ', 'заказа', 'заказов')} в партии
              </p>
            </div>
            <Button onClick={onBack} className="shrink-0">
              <Icon name="chevronLeft" size={16} />
              К списку
            </Button>
          </div>
        </section>

        <ErrorBanner error={error} />

        {selectedOrders.length ? (
          <div className="space-y-4">
            <section className={panelClass}>
              <PanelHeader title="Заказы партии" eyebrow="Нажмите заказ, чтобы раскрыть детали" />
              <div className="divide-y divide-stone-100">
                {selectedOrders.map((order) => (
                  <CollapsedOrder
                    key={order.number}
                    order={order}
                    catalog={catalog}
                    expanded={expanded.has(order.number)}
                    onToggle={() => toggle(order.number)}
                    onRemove={() => onRemove(order.number)}
                    onOpenProduction={onOpenProduction}
                    loading={loading}
                  />
                ))}
              </div>
            </section>

            <div className="grid items-start gap-4 lg:grid-cols-2">
              <section className={panelClass}>
                <PanelHeader title="Отработка" />
                <p className="m-0 mb-2 text-note text-stone-500">
                  Укажите закладку и фактический выход. Заявки не изменяются; значения разносятся по заказам.
                </p>
                <ProductionSheet
                  key={sheetKey}
                  orders={selectedOrders}
                  loading={loading}
                  onSave={onSaveProduction}
                  onOpenJournal={onOpenJournal}
                  onDirtyChange={setProductionDirty}
                />
              </section>
              <section className={panelClass}>
                <PanelHeader title="Расчёт теста" />
                {productionDirty && (
                  <p className="m-0 mb-2 rounded-md border border-amber-200 bg-amber-50 px-3 py-2 text-note text-amber-800" role="status">
                    Сначала сохраните или отмените правки отработки — расчёт идёт по сохранённому факту.
                  </p>
                )}
                <MonitorReports
                  monitor={productionDirty ? null : monitor}
                  onCalculate={onCalculate}
                  loading={loading}
                  canCalculate={selectedOrders.length > 0 && !productionDirty}
                />
              </section>
            </div>
          </div>
        ) : (
          <EmptyState>Выбор пустой.</EmptyState>
        )}
      </div>
    </section>
  );
}
