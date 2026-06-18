import { Button } from '../../components/Button';
import { EmptyState } from '../../components/EmptyState';
import { panelClass, PanelHeader } from '../../components/Panel';
import { MonitorReports } from './MonitorReports';
import { OrderDetails } from './OrderDetails';

export function BakerSelectionReview({
  loading,
  selectedOrders,
  monitor,
  error,
  onBack,
  onRemove,
  onCalculate,
}) {
  return (
    <section className="px-3 py-3 sm:px-5 lg:px-6">
      <div className="mx-auto max-w-[1440px] space-y-4">
        <section className="rounded-lg border border-stone-300 bg-[#fff7df] p-3">
          <div className="flex flex-wrap items-center justify-between gap-2">
            <div>
              <h1 className="m-0 text-[18px] font-semibold leading-7 text-stone-950">Выбранные заказы</h1>
              <p className="m-0 text-[13px] leading-5 text-stone-600">Заказов: {selectedOrders.length}</p>
            </div>
            <div className="flex flex-wrap gap-2">
              <Button onClick={onBack}>К списку</Button>
              <Button variant="primary" onClick={onCalculate} disabled={loading || selectedOrders.length === 0}>
                Рассчитать тесто
              </Button>
            </div>
          </div>
        </section>

        {error && <div className="rounded-md border border-red-200 bg-red-50 px-3 py-2 text-[13px] text-red-800">{error}</div>}

        {selectedOrders.length ? (
          <div className="grid gap-4 xl:grid-cols-[minmax(0,1.1fr)_minmax(22rem,0.9fr)]">
            <div className="space-y-4">
              {selectedOrders.map((order) => (
                <section className={panelClass} key={order.number}>
                  <div className="mb-3 flex justify-end">
                    <Button onClick={() => onRemove(order.number)} disabled={loading}>
                      Убрать
                    </Button>
                  </div>
                  <OrderDetails order={order} />
                </section>
              ))}
            </div>
            <section className={panelClass}>
              <PanelHeader title="Расчёт теста" />
              <MonitorReports monitor={monitor} />
            </section>
          </div>
        ) : (
          <EmptyState>Выбор пустой.</EmptyState>
        )}
      </div>
    </section>
  );
}
