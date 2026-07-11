import { useState } from 'react';
import { Button } from '../../ui/Button';
import { CopyButton } from '../../ui/CopyButton';
import { EmptyState } from '../../ui/EmptyState';
import { Icon } from '../../ui/Icon';
import { Modal } from '../../ui/Modal';
import { formatQuantity } from '../../lib/format';
import { monitorToText } from '../../lib/orders';

export function MonitorReports({ monitor, onCalculate, loading, canCalculate = true }) {
  if (!monitor) {
    if (onCalculate) {
      return (
        <div className="mt-3 flex flex-col items-center gap-2 rounded-lg border border-dashed border-stone-300 bg-stone-50 px-4 py-8 text-center">
          <p className="m-0 text-[13px] text-stone-500">Расчёт расхода теста по заказам.</p>
          <Button variant="primary" onClick={onCalculate} loading={loading} disabled={!canCalculate}>
            <Icon name="calculator" size={16} />
            Рассчитать тесто
          </Button>
        </div>
      );
    }
    return <EmptyState compact>Нажмите "Рассчитать", чтобы увидеть расход теста.</EmptyState>;
  }
  if (monitor.total_reports?.length) {
    return (
      <div className="fade-in mt-3 space-y-3">
        <CopyBar monitor={monitor} />
        <ReportSection title={`Итого по заказам: ${monitor.orders?.length || 0}`} reports={monitor.total_reports} />
        {monitor.orders?.length > 1 && (
          <div className="space-y-2">
            <h3 className="m-0 px-0.5 text-[11px] font-medium uppercase tracking-wide text-stone-500">По заказам</h3>
            {monitor.orders.map((order) => (
              <OrderSummaryCard key={order.order?.number} title={order.order?.number || 'Заказ'} reports={order.reports || []} />
            ))}
          </div>
        )}
      </div>
    );
  }
  if (!monitor.reports?.length) {
    return <EmptyState compact>Нет данных для расчёта теста.</EmptyState>;
  }
  return (
    <div className="fade-in mt-3 space-y-3">
      <CopyBar monitor={monitor} />
      <ReportSection reports={monitor.reports} />
    </div>
  );
}

function CopyBar({ monitor }) {
  const [detailsOpen, setDetailsOpen] = useState(false);
  return (
    <div className="flex justify-end gap-2">
      <Button onClick={() => setDetailsOpen(true)}>
        <Icon name="orders" size={15} />
        Расшифровка
      </Button>
      <CopyButton getText={() => monitorToText(monitor)} label="Копировать расчёт" />
      {detailsOpen && <MonitorDetailsModal monitor={monitor} onClose={() => setDetailsOpen(false)} />}
    </div>
  );
}

// MonitorDetailsModal — popup с полной расшифровкой расчёта: по каждому тесту
// итог, раскладка по блюдам (по эффективному количеству с учётом отработки)
// и состав по техкарте; в batch-режиме — итоги и каждый заказ отдельно.
function MonitorDetailsModal({ monitor, onClose }) {
  const isBatch = Boolean(monitor.total_reports?.length);
  return (
    <Modal onClose={onClose}>
      <div className="mb-3 flex items-center justify-between gap-2">
        <h3 className="m-0 text-[16px] font-semibold leading-6 text-stone-950">Расшифровка расчёта</h3>
        <Button onClick={onClose}>
          <Icon name="close" size={16} />
          Закрыть
        </Button>
      </div>
      <div className="space-y-4">
        {isBatch ? (
          <>
            <DetailsSection title={`Итого по заказам: ${monitor.orders?.length || 0}`} reports={monitor.total_reports} />
            {monitor.orders?.map((order) => (
              <DetailsSection key={order.order?.number} title={`Заказ ${order.order?.number || ''}`} reports={order.reports || []} />
            ))}
          </>
        ) : (
          <DetailsSection reports={monitor.reports || []} />
        )}
      </div>
    </Modal>
  );
}

function DetailsSection({ title = '', reports }) {
  const used = (reports || []).filter(({ report }) => report?.ingredient?.quantity > 0);
  if (!used.length) return null;
  return (
    <section className="space-y-2">
      {title && <h4 className="m-0 text-[12px] font-semibold uppercase tracking-wide text-stone-500">{title}</h4>}
      {used.map(({ code, report }) => (
        <article className="overflow-hidden rounded-lg border border-stone-200" key={code}>
          <header className="flex items-baseline justify-between gap-3 border-b border-stone-200 bg-stone-50 px-3 py-2">
            <strong className="min-w-0 break-words text-[13px] font-semibold leading-5 text-stone-950">{report.ingredient.product_name}</strong>
            <span className="shrink-0 text-[14px] font-bold leading-5 tabular-nums text-stone-950">
              {formatQuantity(report.ingredient.quantity)} <span className="text-[12px] font-medium text-stone-500">{report.ingredient.unit}</span>
            </span>
          </header>
          {report.breakdown?.length > 0 && (
            <div>
              <p className="m-0 bg-stone-50/60 px-3 py-1 text-[11px] font-medium uppercase tracking-wide text-stone-400">По блюдам</p>
              <ul className="m-0 list-none divide-y divide-stone-100 p-0">
                {report.breakdown.map((item) => (
                  <li className="flex items-baseline justify-between gap-3 px-3 py-1 text-[13px]" key={`${code}-${item.order_item_code}-${item.order_item_name}`}>
                    <span className="min-w-0 break-words leading-5 text-stone-600">
                      {item.order_item_name}
                      <span className="text-stone-400"> × {formatQuantity(item.order_item_quantity)}</span>
                    </span>
                    <span className="shrink-0 leading-5 tabular-nums text-stone-900">
                      {formatQuantity(item.ingredient_quantity)} <span className="text-[12px] text-stone-500">{report.ingredient.unit}</span>
                    </span>
                  </li>
                ))}
              </ul>
            </div>
          )}
          {report.components?.length > 0 && (
            <div className="border-t border-stone-200">
              <p className="m-0 bg-stone-50/60 px-3 py-1 text-[11px] font-medium uppercase tracking-wide text-stone-400">Состав</p>
              <ul className="m-0 list-none divide-y divide-stone-100 p-0">
                {report.components.map((item, index) => (
                  <li className="flex items-baseline justify-between gap-3 px-3 py-1 text-[13px]" key={`${code}-c-${item.product_code || index}`}>
                    <span className="min-w-0 break-words leading-5 text-stone-600">{item.product_name}</span>
                    <span className="shrink-0 leading-5 tabular-nums text-stone-900">
                      {formatQuantity(item.quantity)} <span className="text-[12px] text-stone-500">{item.unit}</span>
                    </span>
                  </li>
                ))}
              </ul>
            </div>
          )}
        </article>
      ))}
    </section>
  );
}

function ReportSection({ title = '', reports }) {
  const used = reports.filter(({ report }) => report?.ingredient?.quantity > 0);
  const unused = reports.filter(({ report }) => !(report?.ingredient?.quantity > 0));
  return (
    <section className="space-y-2">
      {title && <h3 className="m-0 px-0.5 text-[11px] font-medium uppercase tracking-wide text-stone-500">{title}</h3>}
      {used.map(({ code, report }) => (
        <DoughCard key={code} report={report} />
      ))}
      {used.length === 0 && <EmptyState compact>Тесто в этих заказах не используется.</EmptyState>}
      {unused.length > 0 && used.length > 0 && (
        <p className="m-0 px-0.5 text-[12px] leading-5 text-stone-400">
          Не используется: {unused.map(({ report }) => report.ingredient.product_name).join(', ')}
        </p>
      )}
    </section>
  );
}

// DoughCard shows one dough: total on top, its tech-card ingredients below.
function DoughCard({ report }) {
  const ing = report.ingredient;
  return (
    <article className="overflow-hidden rounded-xl border border-stone-200 bg-white shadow-sm">
      <header className="flex items-baseline justify-between gap-3 border-b border-stone-200 bg-stone-50 px-3 py-2">
        <strong className="min-w-0 break-words text-[13px] font-semibold leading-5 text-stone-950">{ing.product_name}</strong>
        <span className="shrink-0 text-[16px] font-bold leading-5 tabular-nums text-stone-950">
          {formatQuantity(ing.quantity)} <span className="text-[12px] font-medium text-stone-500">{ing.unit}</span>
        </span>
      </header>
      {report.components?.length > 0 && (
        <ul className="m-0 list-none divide-y divide-stone-100 p-0">
          {report.components.map((item, index) => (
            <li
              key={item.product_code || index}
              className="flex items-baseline justify-between gap-3 px-3 py-1.5 text-[13px]"
            >
              <span className="min-w-0 break-words leading-5 text-stone-600">{item.product_name}</span>
              <span className="shrink-0 leading-5 tabular-nums text-stone-900">
                {formatQuantity(item.quantity)} <span className="text-[12px] text-stone-500">{item.unit}</span>
              </span>
            </li>
          ))}
        </ul>
      )}
    </article>
  );
}

// OrderSummaryCard is the compact per-order block in a batch calculation:
// order number + dough totals, without the ingredient details.
function OrderSummaryCard({ title, reports }) {
  const used = reports.filter(({ report }) => report?.ingredient?.quantity > 0);
  return (
    <article className="overflow-hidden rounded-xl border border-stone-200 bg-white shadow-sm">
      <h4 className="m-0 border-b border-stone-200 bg-stone-50 px-3 py-1.5 text-[13px] font-semibold leading-5 text-stone-950">{title}</h4>
      {used.length > 0 ? (
        <ul className="m-0 list-none divide-y divide-stone-100 p-0">
          {used.map(({ code, report }) => (
            <li key={code} className="flex items-baseline justify-between gap-3 px-3 py-1.5 text-[13px]">
              <span className="min-w-0 break-words leading-5 text-stone-600">{report.ingredient.product_name}</span>
              <span className="shrink-0 font-semibold leading-5 tabular-nums text-stone-950">
                {formatQuantity(report.ingredient.quantity)} <span className="text-[12px] font-medium text-stone-500">{report.ingredient.unit}</span>
              </span>
            </li>
          ))}
        </ul>
      ) : (
        <p className="m-0 px-3 py-1.5 text-[12px] leading-5 text-stone-400">Тесто не используется.</p>
      )}
    </article>
  );
}
