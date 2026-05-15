import { useEffect, useMemo, useState } from 'react';
import { Button } from '../../components/Button';
import { EmptyState } from '../../components/EmptyState';
import { panelClass, PanelHeader } from '../../components/Panel';
import { apiBase, buildMode, frontendLogsEnabled } from '../../config/env';
import { fetchOrder, fetchOrderMonitor, fetchOrders } from '../../api/orders';
import { logInfo, logWarn } from '../../lib/logger';
import { orderNumberFromLocation, syncOrderURL, trimString } from '../../lib/url';
import { MonitorReports } from './MonitorReports';
import { OrderDetails } from './OrderDetails';
import { OrderList } from './OrderList';

const defaultPage = {
  page: 1,
  limit: 10,
  total: 0,
  total_pages: 0,
};

export function OrdersPage() {
  const [orders, setOrders] = useState([]);
  const [selectedOrder, setSelectedOrder] = useState(null);
  const [monitor, setMonitor] = useState(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');
  const [ordersPage, setOrdersPage] = useState(defaultPage);

  const selectedNumber = selectedOrder?.number || '';
  const pageTitle = useMemo(() => selectedNumber || 'Последние заказы', [selectedNumber]);

  useEffect(() => {
    const linkedOrderNumber = orderNumberFromLocation();
    logInfo('app.config', {
      mode: buildMode,
      api_base: apiBase,
      frontend_logs: frontendLogsEnabled,
      page_origin: window.location.origin,
      page_path: window.location.pathname,
      page_search: window.location.search,
      linked_order: linkedOrderNumber,
    });
    loadOrders(ordersPage.page, linkedOrderNumber);
    if (linkedOrderNumber) {
      loadOrder(linkedOrderNumber);
    }
  }, []);

  async function run(task) {
    setLoading(true);
    setError('');
    try {
      await task();
    } catch (err) {
      logWarn('action.failed', {
        error: err instanceof Error ? err.message : String(err),
      });
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setLoading(false);
    }
  }

  function loadOrders(page = ordersPage.page, linkedOrderNumber = '') {
    return run(async () => {
      const result = await fetchOrders(page, ordersPage.limit);
      const items = result.items || [];
      logInfo('orders.loaded', {
        page: result.page || page,
        limit: result.limit || ordersPage.limit,
        total: result.total || 0,
        count: items.length,
      });
      setOrders(items);
      setOrdersPage({
        page: result.page || page,
        limit: result.limit || ordersPage.limit,
        total: result.total || 0,
        total_pages: result.total_pages || 0,
      });
      if (!selectedOrder && items.length > 0 && !trimString(linkedOrderNumber)) {
        setSelectedOrder(items[0]);
      }
    });
  }

  function loadOrder(number) {
    return run(async () => {
      syncOrderURL(number);
      const order = await fetchOrder(number);
      logInfo('order.loaded', {
        number,
        items: order.items?.length || 0,
      });
      setSelectedOrder(order);
      setMonitor(null);
    });
  }

  function loadMonitor() {
    if (!selectedOrder) return;
    return run(async () => {
      const result = await fetchOrderMonitor(selectedOrder.number);
      logInfo('dough_calculation.loaded', {
        order_number: selectedOrder.number,
        reports: result.reports?.length || 0,
      });
      setMonitor(result);
    });
  }

  return (
    <main className="min-h-screen bg-[#f7f7f5] text-stone-900 lg:grid lg:grid-cols-[24rem_minmax(0,1fr)]">
      <OrderList
        loading={loading}
        orders={orders}
        page={ordersPage}
        selectedNumber={selectedNumber}
        onRefresh={() => loadOrders(ordersPage.page)}
        onSelect={loadOrder}
        onPageChange={loadOrders}
      />
      <section className="min-w-0 p-3 pt-0 sm:p-5 lg:p-6">
        <div className="mx-auto max-w-[1180px]">
          {error && <div className="mb-4 rounded-md border border-red-200 bg-red-50 px-3 py-2 text-[13px] text-red-800">{error}</div>}
          {selectedOrder ? (
            <div className="grid gap-4 xl:grid-cols-[minmax(0,1.1fr)_minmax(22rem,0.9fr)]">
              <section className={panelClass}>
                <OrderDetails order={selectedOrder} />
              </section>
              <section className={panelClass}>
                <PanelHeader title="Расчёт теста" />
                <Button variant="primary" onClick={loadMonitor} disabled={!selectedOrder || loading}>
                  Рассчитать
                </Button>
                <MonitorReports monitor={monitor} />
              </section>
            </div>
          ) : (
            <EmptyState>Заказы не загружены.</EmptyState>
          )}
        </div>
      </section>
    </main>
  );
}
