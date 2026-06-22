import { useEffect, useRef, useState } from 'react';
import { Login } from '../../components/Login';
import { MePanel } from '../account/Me';
import { AdminUsers } from '../admin/AdminUsers';
import { AdminDishes } from '../admin/AdminDishes';
import { apiBase, buildMode, frontendLogsEnabled } from '../../config/env';
import { ApiError } from '../../api/client';
import { createOrder, fetchBatchOrderMonitor, fetchCatalog, fetchDepartments, fetchMe, fetchOrder, fetchOrderMonitor, fetchOrders, setOrderFavorite, updateOrder } from '../../api/orders';
import { clearToken, getToken, isWebMode } from '../../lib/auth';
import { logInfo, logWarn } from '../../lib/logger';
import { miniAppModeFromLocation, orderNumberFromLocation, orderNumbersFromLocation, trimString } from '../../lib/url';
import { OrdersLayout } from './OrdersLayout';
import { BakerOrdersView } from './BakerOrdersView';
import { BakerSelectionReview } from './BakerSelectionReview';
import { BakerOrderReview } from './BakerOrderReview';
import { ShopOrdersView } from './ShopOrdersView';

const defaultPage = {
  page: 1,
  limit: 10,
  total: 0,
  total_pages: 0,
};

export function OrdersPage({ route = { name: 'orders' }, navigate = () => {} }) {
  const [orders, setOrders] = useState([]);
  const [shops, setShops] = useState([]);
  const [catalog, setCatalog] = useState([]);
  const [viewer, setViewer] = useState(null);
  const [editor, setEditor] = useState(null);
  const [selectedOrder, setSelectedOrder] = useState(null);
  const [selectedOrderNumbers, setSelectedOrderNumbers] = useState([]);
  const [selectedOrders, setSelectedOrders] = useState([]);
  const [selectionMode, setSelectionMode] = useState(false);
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

  function bootstrap(activeRoute = route) {
    const launchMode = miniAppModeFromLocation();
    const linkedOrderNumber = orderNumberFromLocation();
    const linkedOrderNumbers = orderNumbersFromLocation();
    const routeOrderNumber = activeRoute.number || '';
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
      route: activeRoute.name,
      route_order: routeOrderNumber,
    });
    loadViewer(launchMode, linkedOrderNumber, activeRoute);
    loadOrders(ordersPage.page, routeOrderNumber || linkedOrderNumber);
    if (activeRoute.name === 'orderSelection') {
      loadSelectedOrders();
    } else if (launchMode === 'monitor' && linkedOrderNumbers.length) {
      loadMonitor(linkedOrderNumbers, true);
    } else if (launchMode === 'monitor' && linkedOrderNumber) {
      loadOrder(linkedOrderNumber, true);
    } else if (activeRoute.name !== 'orderEdit' && routeOrderNumber) {
      loadOrder(routeOrderNumber, activeRoute.name === 'orderMonitor', false);
    } else if (linkedOrderNumber && launchMode !== 'edit') {
      loadOrder(linkedOrderNumber, false, false);
    } else if (activeRoute.name === 'orderNew') {
      openCreateOrder(false);
    }
  }

  useEffect(() => {
    if (authNeeded) return;
    bootstrap(route);
  }, [authNeeded, route.name, route.number]);

  function handleAuthenticated() {
    setAuthNeeded(false);
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
      return await task();
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

  function loadViewer(launchMode = '', linkedOrderNumber = '', activeRoute = route) {
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
        if (activeRoute.name === 'orderNew' || launchMode === 'create') {
          setEditor({ mode: 'create', order: null });
        } else if (activeRoute.name === 'orderEdit' || (launchMode === 'edit' && linkedOrderNumber)) {
          const order = await fetchOrder(activeRoute.number || linkedOrderNumber);
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
      setSelectedOrders([]);
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
      setSelectedOrders([]);
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

  function loadOrder(number, withMonitor = false, pushRoute = true) {
    return run(async () => {
      if (pushRoute) {
        navigate({ name: withMonitor ? 'orderMonitor' : 'orderView', number });
      }
      const order = await fetchOrder(number);
      logInfo('order.loaded', {
        number,
        items: order.items?.length || 0,
      });
      setSelectedOrder(order);
      setEditor(null);
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
      const exists = current.includes(number);
      if (exists) {
        setSelectedOrders((orders) => orders.filter((order) => order.number !== number));
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

  function calculateSelectedOrders() {
    return loadMonitor(selectedOrderNumbers);
  }

  function calculateCurrentOrder() {
    if (!selectedOrder?.number) return;
    return loadMonitor([selectedOrder.number]);
  }

  function loadSelectedOrders() {
    if (selectedOrderNumbers.length === 0) {
      setSelectedOrders([]);
      setMonitor(null);
      return;
    }
    return run(async () => {
      const loaded = await Promise.all(selectedOrderNumbers.map((number) => fetchOrder(number)));
      setSelectedOrders(loaded);
    });
  }

  function openSelectionReview() {
    return run(async () => {
      const loaded = await Promise.all(selectedOrderNumbers.map((number) => fetchOrder(number)));
      setSelectedOrders(loaded);
      setSelectionMode(false);
      navigate({ name: 'orderSelection' });
    });
  }

  function removeSelectedOrder(number) {
    setSelectedOrderNumbers((current) => current.filter((item) => item !== number));
    setSelectedOrders((current) => current.filter((order) => order.number !== number));
    setMonitor(null);
  }

  function openCreateOrder(pushRoute = true) {
    if (pushRoute) {
      navigate({ name: 'orderNew' });
    }
    setEditor({ mode: 'create', order: null });
    setMonitor(null);
  }

  function openUpdateOrder() {
    if (!selectedOrder) return;
    navigate({ name: 'orderEdit', number: selectedOrder.number });
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
      navigate({ name: 'orderView', number: saved.number });
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

  function toggleFavorite(number, favorite) {
    return run(async () => {
      const saved = await setOrderFavorite(number, favorite);
      setSelectedOrder((current) => (current && current.number === number ? saved : current));
      setOrders((current) => current.map((o) => (o.number === number ? { ...o, favorite: saved.favorite } : o)));
    });
  }

  const canFavorite = viewer?.role === 'admin';
  const canUseMonitor = viewer?.department_type === 'workshop' || viewer?.role === 'baker';
  // Only honour the editor on its own routes; navigating away (e.g. nav "Заказы"
  // from create) must fall back to the list/details view even if editor state
  // lingers.
  const editorRoute = route.name === 'orderNew' || route.name === 'orderEdit';
  const activeEditor = editorRoute ? editor : null;
  const showCreateOrderPage = activeEditor?.mode === 'create';

  if (authNeeded) {
    return <Login onAuthenticated={handleAuthenticated} />;
  }

  if (route.name === 'me') {
    return (
      <OrdersLayout viewer={viewer} active={route.name} onNavigate={navigate} onLogout={handleLogout}>
        <MePanel viewer={viewer} onLogout={handleLogout} />
      </OrdersLayout>
    );
  }

  if (viewer?.role === 'admin' && route.name === 'adminUsers') {
    return (
      <OrdersLayout viewer={viewer} active={route.name} onNavigate={navigate} onLogout={handleLogout}>
        <AdminUsers />
      </OrdersLayout>
    );
  }

  if (viewer?.role === 'admin' && route.name === 'adminDishes') {
    return (
      <OrdersLayout viewer={viewer} active={route.name} onNavigate={navigate} onLogout={handleLogout}>
        <AdminDishes />
      </OrdersLayout>
    );
  }

  return (
    <OrdersLayout viewer={viewer} active={route.name} onNavigate={navigate} onLogout={handleLogout}>
      {canUseMonitor && route.name === 'orderSelection' ? (
        <BakerSelectionReview
          loading={loading}
          selectedOrders={selectedOrders}
          monitor={monitor}
          error={error}
          onBack={() => navigate({ name: 'orders' })}
          onRemove={removeSelectedOrder}
          onCalculate={calculateSelectedOrders}
        />
      ) : canUseMonitor && (route.name === 'orderView' || route.name === 'orderMonitor') ? (
        <BakerOrderReview
          loading={loading}
          order={selectedOrder}
          monitor={monitor}
          error={error}
          canFavorite={canFavorite}
          onToggleFavorite={toggleFavorite}
          onBack={() => navigate({ name: 'orders' })}
          onCalculate={calculateCurrentOrder}
        />
      ) : canUseMonitor ? (
        <BakerOrdersView
          loading={loading}
          orders={orders}
          page={ordersPage}
          shops={shops}
          filters={filters}
          selectedNumber={selectedNumber}
          selectedOrder={selectedOrder}
          selectedOrderNumbers={selectedOrderNumbers}
          error={error}
          selectionMode={selectionMode}
          canFavorite={canFavorite}
          onToggleFavorite={toggleFavorite}
          onLoadOrder={fetchOrder}
          onSelect={(number) => loadOrder(number, false)}
          onToggleSelection={toggleOrderSelection}
          onToggleSelectionMode={() => setSelectionMode((current) => !current)}
          onOpenSelection={openSelectionReview}
          onPageChange={loadOrders}
          onFiltersChange={updateFilters}
          onResetFilters={resetFilters}
        />
      ) : (
        <ShopOrdersView
          loading={loading}
          orders={orders}
          page={ordersPage}
          shops={shops}
          viewer={viewer}
          filters={filters}
          selectedNumber={selectedNumber}
          selectedOrder={selectedOrder}
          editor={activeEditor}
          catalog={catalog}
          error={error}
          showCreateOrderPage={showCreateOrderPage}
          canFavorite={canFavorite}
          onToggleFavorite={toggleFavorite}
          onSelect={(number) => loadOrder(number)}
          onPageChange={loadOrders}
          onFiltersChange={updateFilters}
          onResetFilters={resetFilters}
          onEdit={openUpdateOrder}
          onCancelEdit={() => {
            setEditor(null);
            navigate(editor?.order?.number ? { name: 'orderView', number: editor.order.number } : { name: 'orders' });
          }}
          onSave={saveOrder}
        />
      )}
    </OrdersLayout>
  );
}
