import { useEffect, useRef, useState } from 'react';
import { Button } from '../../components/Button';
import { EmptyState } from '../../components/EmptyState';
import { Login } from '../../components/Login';
import { AdminUsers } from '../admin/AdminUsers';
import { panelClass, PanelHeader } from '../../components/Panel';
import { apiBase, buildMode, frontendLogsEnabled } from '../../config/env';
import { ApiError } from '../../api/client';
import { createOrder, fetchBatchOrderMonitor, fetchCatalog, fetchDepartments, fetchMe, fetchOrder, fetchOrderMonitor, fetchOrders, updateOrder } from '../../api/orders';
import { clearToken, getToken, isWebMode } from '../../lib/auth';
import { logInfo, logWarn } from '../../lib/logger';
import { miniAppModeFromLocation, orderNumberFromLocation, orderNumbersFromLocation, syncOrderURL, trimString } from '../../lib/url';
import { MonitorReports } from './MonitorReports';
import { OrderDetails } from './OrderDetails';
import { OrderEditor } from './OrderEditor';
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
  const [catalog, setCatalog] = useState([]);
  const [viewer, setViewer] = useState(null);
  const [editor, setEditor] = useState(null);
  const [selectedOrder, setSelectedOrder] = useState(null);
  const [selectedOrderNumbers, setSelectedOrderNumbers] = useState([]);
  const [monitor, setMonitor] = useState(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');
  const [authNeeded, setAuthNeeded] = useState(isWebMode() && !getToken());
  const [ordersPage, setOrdersPage] = useState(defaultPage);
  const [filters, setFilters] = useState({
    fromDepartmentID: '',
    fulfillmentDate: '',
  });
  const filtersRef = useRef(filters);

  const selectedNumber = selectedOrder?.number || '';
  const selectedOrderCount = selectedOrderNumbers.length;

  function bootstrap() {
    const launchMode = miniAppModeFromLocation();
    const linkedOrderNumber = orderNumberFromLocation();
    const linkedOrderNumbers = orderNumbersFromLocation();
    logInfo('app.config', {
      mode: buildMode,
      api_base: apiBase,
      frontend_logs: frontendLogsEnabled,
      page_origin: window.location.origin,
      page_path: window.location.pathname,
      page_search: window.location.search,
      launch_mode: launchMode,
      linked_order: linkedOrderNumber,
      linked_orders: linkedOrderNumbers,
    });
    loadViewer(launchMode, linkedOrderNumber);
    loadOrders(ordersPage.page, linkedOrderNumber);
    if (launchMode === 'monitor' && linkedOrderNumbers.length) {
      loadMonitor(linkedOrderNumbers, true);
    } else if (launchMode === 'monitor' && linkedOrderNumber) {
      loadOrder(linkedOrderNumber, true);
    } else if (linkedOrderNumber && launchMode !== 'edit') {
      loadOrder(linkedOrderNumber);
    }
  }

  useEffect(() => {
    if (authNeeded) return;
    bootstrap();
  }, []);

  function handleAuthenticated() {
    setAuthNeeded(false);
    bootstrap();
  }

  function handleLogout() {
    clearToken();
    setViewer(null);
    setAuthNeeded(true);
  }

  async function run(task) {
    setLoading(true);
    setError('');
    try {
      await task();
    } catch (err) {
      if (err instanceof ApiError && err.status === 401 && isWebMode()) {
        clearToken();
        setAuthNeeded(true);
        return;
      }
      logWarn('action.failed', {
        error: err instanceof Error ? err.message : String(err),
      });
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setLoading(false);
    }
  }

  function loadViewer(launchMode = '', linkedOrderNumber = '') {
    return run(async () => {
      const current = await fetchMe();
      setViewer(current);
      if (current.department_type === 'workshop') {
        const result = await fetchDepartments('shop');
        const items = Array.isArray(result) ? result : [];
        setShops(items);
        logInfo('shops.loaded', { count: items.length });
      } else if (current.department_type === 'shop') {
        const items = await fetchCatalog();
        setCatalog(Array.isArray(items) ? items : []);
        if (launchMode === 'create') {
          setEditor({ mode: 'create', order: null });
        } else if (launchMode === 'edit' && linkedOrderNumber) {
          const order = await fetchOrder(linkedOrderNumber);
          setSelectedOrder(order);
          setEditor({ mode: 'update', order });
        }
      }
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

  function loadOrder(number, withMonitor = false) {
    return run(async () => {
      syncOrderURL(number, withMonitor ? 'monitor' : 'view');
      const order = await fetchOrder(number);
      logInfo('order.loaded', {
        number,
        items: order.items?.length || 0,
      });
      setSelectedOrder(order);
      if (withMonitor) {
        setMonitor(await fetchOrderMonitor(number));
      } else {
        setMonitor(null);
      }
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

  function loadMonitor(numbers, selectOrders = false) {
    if (!numbers.length) return;
    return run(async () => {
      const result = numbers.length === 1 ? await fetchOrderMonitor(numbers[0]) : await fetchBatchOrderMonitor(numbers);
      logInfo('dough_batch_calculation.loaded', {
        orders: numbers,
        selected_count: numbers.length,
        reports: result.total_reports?.length || result.reports?.length || 0,
      });
      if (selectOrders) {
        setSelectedOrderNumbers(numbers);
      }
      setMonitor(result);
    });
  }

  function loadBatchMonitor() {
    const numbers = selectedOrderNumbers.length ? selectedOrderNumbers : selectedOrder ? [selectedOrder.number] : [];
    return loadMonitor(numbers);
  }

  function openCreateOrder() {
    setEditor({ mode: 'create', order: null });
    setMonitor(null);
  }

  function openUpdateOrder() {
    if (!selectedOrder) return;
    setEditor({ mode: 'update', order: selectedOrder });
    setMonitor(null);
  }

  function saveOrder(request) {
    return run(async () => {
      const saved = editor?.mode === 'update'
        ? await updateOrder(editor.order.number, request)
        : await createOrder(request);
      setSelectedOrder(saved);
      setEditor(null);
      syncOrderURL(saved.number);
      const result = await fetchOrders(1, ordersPage.limit, filtersRef.current);
      const items = Array.isArray(result.items) ? result.items : [];
      setOrders(items);
      setOrdersPage({
        page: result.page || 1,
        limit: result.limit || ordersPage.limit,
        total: result.total || 0,
        total_pages: result.total_pages || 0,
      });
    });
  }

  const canWriteOrders = viewer?.department_type === 'shop';

  if (authNeeded) {
    return <Login onAuthenticated={handleAuthenticated} />;
  }

  if (viewer?.role === 'admin') {
    return <AdminUsers onLogout={handleLogout} />;
  }

  return (
    <main className="min-h-screen bg-[#fff7df] text-stone-900 lg:grid lg:grid-cols-[24rem_minmax(0,1fr)]">
      <OrderList
        loading={loading}
        orders={orders}
        page={ordersPage}
        shops={shops}
        viewer={viewer}
        canFilterShops={viewer?.department_type === 'workshop'}
        canWriteOrders={canWriteOrders}
        filters={filters}
        selectedNumber={selectedNumber}
        selectedOrderNumbers={selectedOrderNumbers}
        onRefresh={() => loadOrders(ordersPage.page)}
        onCreate={openCreateOrder}
        onSelect={loadOrder}
        onToggleSelection={toggleOrderSelection}
        onPageChange={loadOrders}
        onFiltersChange={updateFilters}
        onResetFilters={resetFilters}
      />
      <section className="min-w-0 p-3 pt-0 sm:p-5 lg:p-6">
        <div className="mx-auto max-w-[1180px]">
          <div className="mb-3 flex flex-wrap justify-end gap-2 pt-3">
            <a href="/me" className="rounded-md border border-stone-300 px-3 py-1.5 text-sm">
              Профиль{viewer?.role ? ` (${viewer.role})` : ''}
            </a>
            {isWebMode() && (
              <button onClick={handleLogout} className="rounded-md border border-red-300 px-3 py-1.5 text-sm font-medium text-red-700">
                Выйти
              </button>
            )}
          </div>
          {error && <div className="mb-4 rounded-md border border-red-200 bg-red-50 px-3 py-2 text-[13px] text-red-800">{error}</div>}
          {editor ? (
            <section className={panelClass}>
              <OrderEditor
                key={`${editor.mode}-${editor.order?.number || 'new'}`}
                catalog={catalog}
                order={editor.order}
                loading={loading}
                onCancel={() => setEditor(null)}
                onSave={saveOrder}
              />
            </section>
          ) : selectedOrder ? (
            <div className="grid gap-4 xl:grid-cols-[minmax(0,1.1fr)_minmax(22rem,0.9fr)]">
              <section className={panelClass}>
                {canWriteOrders && (
                  <div className="mb-3 flex justify-end">
                    <Button onClick={openUpdateOrder} disabled={loading}>
                      Изменить
                    </Button>
                  </div>
                )}
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
