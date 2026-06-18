import { Button } from '../../components/Button';
import { EmptyState } from '../../components/EmptyState';
import { Icon } from '../../components/Icon';
import { panelClass, PanelHeader } from '../../components/Panel';
import { MonitorReports } from './MonitorReports';
import { OrderDetails } from './OrderDetails';

export function BakerOrderReview({ loading, order, monitor, error, onBack, onCalculate }) {
  return (
    <section className="px-3 py-3 pb-20 sm:px-5 sm:pb-3 lg:px-6">
      <div className="mx-auto max-w-[1440px] space-y-4">
        <section className="rounded-lg border border-stone-300 bg-white p-3 shadow-sm">
          <div className="flex flex-wrap items-center justify-between gap-2">
            <div>
              <h1 className="m-0 text-[18px] font-semibold leading-7 text-stone-950">{order?.number || 'Заказ'}</h1>
              <p className="m-0 text-[13px] leading-5 text-stone-600">Просмотр и расчёт</p>
            </div>
            <Button onClick={onBack} className="shrink-0">
              <Icon name="chevronLeft" size={16} />
              К списку
            </Button>
          </div>
        </section>

        {error && <div className="rounded-md border border-red-200 bg-red-50 px-3 py-2 text-[13px] text-red-800">{error}</div>}

        {order ? (
          <div className="grid gap-4 xl:grid-cols-[minmax(0,1.1fr)_minmax(22rem,0.9fr)]">
            <section className={panelClass}>
              <OrderDetails order={order} />
            </section>
            <section className={panelClass}>
              <PanelHeader title="Расчёт теста" />
              <MonitorReports monitor={monitor} onCalculate={onCalculate} loading={loading} canCalculate={Boolean(order)} />
            </section>
          </div>
        ) : (
          <EmptyState>Заказ не выбран.</EmptyState>
        )}
      </div>
    </section>
  );
}
