import { useEffect, useRef, useState } from 'react';
import { Button } from '../../components/Button';
import { EmptyState } from '../../components/EmptyState';
import { panelClass, PanelHeader } from '../../components/Panel';
import { apiBase, buildMode, frontendLogsEnabled } from '../../config/env';
import { fetchBatchOrderMonitor, fetchDepartments, fetchOrder, fetchOrderMonitor, fetchOrders } from '../../api/orders';
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
  const [shops, setShops] = useState([]);
  const [selectedOrder, setSelectedOrder] = useState(null);
  const [selectedOrderNumbers, setSelectedOrderNumbers] = useState([]);
  const [monitor, setMonitor] = useState(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');
  const [ordersPage, setOrdersPage] = useState(defaultPage);
  const [filters, setFilters] = useState({
    fromDepartmentID: '',
    fulfillmentDate: '',
  });
  const filtersRef = useRef(filters);

  const selectedNumber = selectedOrder?.number || '';
  const selectedOrderCount = selectedOrderNumbers.length;

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
    loadShops();
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

  function loadShops() {
    return run(async () => {
      const result = await fetchDepartments('shop');
      const items = Array.isArray(result) ? result : [];
      setShops(items);
      logInfo('shops.loaded', { count: items.length });
    });
  }

  function loadOrders(page = ordersPage.page, linkedOrderNumber = '', activeFilters = filtersRef.current) {
    return run(async () => {
      const result = await fetchOrders(page, ordersPage.limit, activeFilters);
      const items = Array.isArray(result.items) ? result.items : [];
      logInfo('orders.loaded', {
        page: result.page || page,
        limit: result.limit || ordersPage.limit,
        total: result.total || 0,
        count: items.length,
        filters: activeFilters,
      });
      setOrders(items);
      setSelectedOrderNumbers((current) => current.filter((number) => items.some((order) => order.number === number)));
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

  function updateFilters(next) {
    const updated = { ...filtersRef.current, ...next };
    filtersRef.current = updated;
    setFilters(updated);
    setMonitor(null);
    return run(async () => {
      const result = await fetchOrders(1, ordersPage.limit, updated);
      const items = Array.isArray(result.items) ? result.items : [];
      setOrders(items);
      setSelectedOrderNumbers([]);
      setOrdersPage({
        page: result.page || 1,
        limit: result.limit || ordersPage.limit,
        total: result.total || 0,
        total_pages: result.total_pages || 0,
      });
      setSelectedOrder(items[0] || null);
      logInfo('orders.filtered', {
        count: items.length,
        filters: updated,
      });
    });
  }

  function resetFilters() {
    const reset = {
      fromDepartmentID: '',
      fulfillmentDate: '',
    };
    filtersRef.current = reset;
    setFilters(reset);
    setMonitor(null);
    return run(async () => {
      const result = await fetchOrders(1, ordersPage.limit, reset);
      const items = Array.isArray(result.items) ? result.items : [];
      setOrders(items);
      setSelectedOrderNumbers([]);
      setOrdersPage({
        page: result.page || 1,
        limit: result.limit || ordersPage.limit,
        total: result.total || 0,
        total_pages: result.total_pages || 0,
      });
      setSelectedOrder(items[0] || null);
      logInfo('orders.filters_reset', {
        count: items.length,
      });
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

  function toggleOrderSelection(number) {
    setMonitor(null);
    setSelectedOrderNumbers((current) => {
      if (current.includes(number)) {
        return current.filter((item) => item !== number);
      }
      return [...current, number];
    });
  }

  function loadBatchMonitor() {
    const numbers = selectedOrderNumbers.length ? selectedOrderNumbers : selectedOrder ? [selectedOrder.number] : [];
    if (!numbers.length) return;
    return run(async () => {
      const result = numbers.length === 1 ? await fetchOrderMonitor(numbers[0]) : await fetchBatchOrderMonitor(numbers);
      logInfo('dough_batch_calculation.loaded', {
        orders: numbers,
        selected_count: numbers.length,
        reports: result.total_reports?.length || result.reports?.length || 0,
      });
      setMonitor(result);
    });
  }

  return (
    <main className="min-h-screen bg-[#fff7df] text-stone-900 lg:grid lg:grid-cols-[24rem_minmax(0,1fr)]">
      <OrderList
        loading={loading}
        orders={orders}
        page={ordersPage}
        shops={shops}
        filters={filters}
        selectedNumber={selectedNumber}
        selectedOrderNumbers={selectedOrderNumbers}
        onRefresh={() => loadOrders(ordersPage.page)}
        onSelect={loadOrder}
        onToggleSelection={toggleOrderSelection}
        onPageChange={loadOrders}
        onFiltersChange={updateFilters}
        onResetFilters={resetFilters}
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
                <div className="flex flex-wrap items-center gap-2">
                  <Button variant="primary" onClick={loadBatchMonitor} disabled={(!selectedOrder && !selectedOrderCount) || loading}>
                    Рассчитать
                  </Button>
                  {selectedOrderCount > 0 && <span className="text-[13px] leading-5 text-stone-600">Выбрано заказов: {selectedOrderCount}</span>}
                </div>
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
