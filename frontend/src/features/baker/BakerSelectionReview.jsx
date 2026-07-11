import { Button } from '../../ui/Button';
import { ErrorBanner } from '../../ui/ErrorBanner';
import { EmptyState } from '../../ui/EmptyState';
import { Icon } from '../../ui/Icon';
import { panelClass, PanelHeader } from '../../ui/Panel';
import { MonitorReports } from './MonitorReports';
import { OrderDetails } from '../orders/OrderDetails';
import { ProductionSheet } from '../production/ProductionSheet';

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
  return (
    <section className="px-3 py-3 pb-20 sm:px-5 sm:pb-3 lg:px-6">
      <div className="mx-auto max-w-[1440px] space-y-4">
        <section className="rounded-lg border border-stone-300 bg-white p-3 shadow-sm">
          <div className="flex flex-wrap items-center justify-between gap-2">
            <div>
              <h1 className="m-0 text-[18px] font-semibold leading-7 text-stone-950">Выбранные заказы</h1>
              <p className="m-0 text-[13px] leading-5 text-stone-600">Заказов: {selectedOrders.length}</p>
            </div>
            <Button onClick={onBack} className="shrink-0">
              <Icon name="chevronLeft" size={16} />
              К списку
            </Button>
          </div>
        </section>

        <ErrorBanner error={error} />

        {selectedOrders.length ? (
          <div className="grid gap-4 xl:grid-cols-[minmax(0,1.1fr)_minmax(22rem,0.9fr)]">
            <div className="space-y-4">
              {selectedOrders.map((order) => (
                <section className={panelClass} key={order.number}>
                  <div className="mb-3 flex justify-end">
                    <Button onClick={() => onRemove(order.number)} disabled={loading}>
                      <Icon name="close" size={16} />
                      Убрать
                    </Button>
                  </div>
                  <OrderDetails order={order} catalog={catalog} onOpenProduction={onOpenProduction} />
                </section>
              ))}
            </div>
            <div className="space-y-4">
              <section className={panelClass}>
                <PanelHeader title="Отработка" />
                <p className="m-0 mb-2 text-[12px] leading-5 text-stone-500">
                  Укажите закладку и фактический выход. Заявки не изменяются; значения разносятся по заказам.
                </p>
                <ProductionSheet
                  orders={selectedOrders}
                  loading={loading}
                  onSave={onSaveProduction}
                  onOpenJournal={onOpenJournal}
                />
              </section>
              <section className={panelClass}>
                <PanelHeader title="Расчёт теста" />
                <MonitorReports monitor={monitor} onCalculate={onCalculate} loading={loading} canCalculate={selectedOrders.length > 0} />
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
