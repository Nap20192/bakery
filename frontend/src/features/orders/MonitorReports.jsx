import { EmptyState } from '../../components/EmptyState';
import { formatQuantity } from '../../lib/format';

export function MonitorReports({ monitor }) {
  if (!monitor) {
    return <EmptyState compact>Нажмите "Рассчитать", чтобы увидеть расход теста.</EmptyState>;
  }
  if (monitor.total_reports?.length) {
    return (
      <div className="mt-3 space-y-3">
        <ReportBlock title={`Итого по заказам: ${monitor.orders?.length || 0}`} reports={monitor.total_reports} />
        {monitor.orders?.map((order) => (
          <ReportBlock key={order.order?.number} title={order.order?.number || 'Заказ'} reports={order.reports || []} compact />
        ))}
      </div>
    );
  }
  if (!monitor.reports?.length) {
    return <EmptyState compact>Нет данных для расчёта теста.</EmptyState>;
  }
  return <ReportBlock reports={monitor.reports} />;
}

function ReportBlock({ title = '', reports, compact = false }) {
  return (
    <div className="overflow-hidden rounded-lg border border-stone-300 bg-white">
      {title && <h3 className="border-b border-stone-300 bg-white px-2.5 py-2 text-[13px] font-semibold leading-5 text-stone-950 sm:px-3">{title}</h3>}
      {reports.map(({ code, report }) => (
        <article className="border-b border-stone-300 last:border-b-0" key={code}>
          <header className="grid grid-cols-[minmax(0,1fr)_auto] gap-2 bg-stone-50 px-2.5 py-1.5 sm:px-3">
            <strong className="min-w-0 break-words text-[13px] font-semibold leading-5 text-stone-950">{report.ingredient.product_name}</strong>
            <span className="text-[13px] font-semibold leading-5 text-stone-950">
              {formatQuantity(report.ingredient.quantity)} {report.ingredient.unit}
            </span>
          </header>
          {!compact && <div className="divide-y divide-stone-300">
            {report.breakdown?.map((item) => (
              <div
                className="grid grid-cols-[minmax(0,1fr)_5.5rem] items-start gap-2 px-2.5 py-1.5 text-[13px] sm:grid-cols-[minmax(0,1fr)_8rem] sm:px-3"
                key={`${code}-${item.order_item_code}`}
              >
                <span className="min-w-0 break-words leading-5 text-stone-700">{item.order_item_name}</span>
                <strong className="text-right text-[12px] font-semibold leading-5 text-stone-950 sm:text-[13px]">
                  {formatQuantity(item.ingredient_quantity)} {report.ingredient.unit}
                </strong>
              </div>
            ))}
          </div>}
        </article>
      ))}
    </div>
  );
}
