import { useEffect, useRef, useState } from 'react';
import { Login } from '../auth/Login';
import { MePanel } from '../account/Me';
import { AdminUsers } from '../admin/AdminUsers';
import { AdminDishes } from '../admin/AdminDishes';
import { apiBase, buildMode, frontendLogsEnabled } from '../../config/env';
import { cancelOrder, createOrder, createProductionSheet, fetchBatchOrderMonitor, fetchCatalog, fetchCategories, fetchDepartments, fetchMe, fetchOrder, fetchOrderMonitor, fetchOrders, restoreOrder, setOrderFavorite, updateOrder } from '../../api/orders';
import { clearToken, getToken, isWebMode } from '../../lib/auth';
import { logInfo } from '../../lib/logger';
import { miniAppModeFromLocation, orderNumberFromLocation, orderNumbersFromLocation, trimString } from '../../lib/url';
import { OrdersLayout } from './OrdersLayout';
import { OrdersTableView } from '../baker/OrdersTableView';
import { ProductionJournal } from '../production/ProductionJournal';
import { BakerOrdersView } from '../baker/BakerOrdersView';
import { BakerSelectionReview } from '../baker/BakerSelectionReview';
import { BakerOrderReview } from '../baker/BakerOrderReview';
import { ShopOrdersView } from '../shop/ShopOrdersView';
import { MATRIX_PAGE_LIMIT, MATRIX_WINDOW_START_OFFSET, matrixWindow } from './orderWindow';
import { useAuthenticatedTask } from './useAuthenticatedTask';

const defaultPage = {
  page: 1,
  limit: 10,
  total: 0,
  total_pages: 0,
};

// canCreateOrders: only the shop role creates/edits orders (and picks the shop
// to send from). Users are no longer bound to a department.
function canCreateOrders(viewer) {
  return viewer?.role === 'shop';
}

export function OrdersPage({ route = { name: 'orders' }, navigate = () => {} }) {
  const [orders, setOrders] = useState([]);
  const [shops, setShops] = useState([]);
  const [catalog, setCatalog] = useState([]);
  const [categories, setCategories] = useState([]);
  const [viewer, setViewer] = useState(null);
  const [editor, setEditor] = useState(null);
  const [selectedOrder, setSelectedOrder] = useState(null);
  const [selectedOrderNumbers, setSelectedOrderNumbers] = useState([]);
  const [selectedOrders, setSelectedOrders] = useState([]);
  const [selectionMode, setSelectionMode] = useState(false);
  const [monitor, setMonitor] = useState(null);
  const [authNeeded, setAuthNeeded] = useState(isWebMode() && !getToken());
  const { error, loading, run, setError } = useAuthenticatedTask(() => setAuthNeeded(true));
  const [ordersPage, setOrdersPage] = useState(defaultPage);
  const [filters, setFilters] = useState({
    fromDepartmentID: '',
    fulfillmentDate: '',
    categoryID: '',
  });
  const filtersRef = useRef(filters);
  const windowStartRef = useRef(MATRIX_WINDOW_START_OFFSET);

  const selectedNumber = selectedOrder?.number || '';

  // shiftMatrixWindow moves the baker's 5-day window earlier/later and reloads.
  function shiftMatrixWindow(deltaDays) {
    windowStartRef.current += deltaDays;
    const windowFilters = { ...filtersRef.current, ...matrixWindow(windowStartRef.current) };
    filtersRef.current = windowFilters;
    setFilters(windowFilters);
    setSelectedOrderNumbers([]);
    setSelectedOrders([]);
    setMonitor(null);
    return loadOrders(1, '', windowFilters, MATRIX_PAGE_LIMIT);
  }

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

  function loadViewer(launchMode = '', linkedOrderNumber = '', activeRoute = route) {
    return run(async () => {
      const current = await fetchMe();
      setViewer(current);
      // Матрица пекаря «магазины × даты» грузится окнами по 5 дней выполнения
      // (вместо постраничной ленты), одной крупной страницей на окно.
      if ((current?.role === 'baker' || current?.role === 'admin') && !filtersRef.current.fulfillmentFrom) {
        const windowFilters = { ...filtersRef.current, ...matrixWindow(windowStartRef.current) };
        filtersRef.current = windowFilters;
        setFilters(windowFilters);
        setOrdersPage((page) => ({ ...page, limit: MATRIX_PAGE_LIMIT }));
        // Номер из URL передаётся дальше, иначе перезагрузка окна перебьёт
        // открытый по прямой ссылке заказ первым заказом списка.
        loadOrders(1, activeRoute.number || linkedOrderNumber, windowFilters, MATRIX_PAGE_LIMIT);
      }
      // Users are no longer bound to a department; behaviour is role-driven.
      // Every role needs the shop list (filtering + the create-order picker).
      const result = await fetchDepartments('shop');
      const shopItems = Array.isArray(result) ? result : [];
      setShops(shopItems);
      logInfo('shops.loaded', { count: shopItems.length });
      // Типы заявок нужны всем ролям: магазину — выбор при создании,
      // пекарю — фильтр и бейджи, админу — управление.
      const categoryItems = await fetchCategories();
      setCategories(Array.isArray(categoryItems) ? categoryItems : []);
      // Каталог тоже нужен всем: магазин собирает по нему заказ, пекарь
      // группирует позиции заказа по группам каталога.
      const items = await fetchCatalog();
      setCatalog(Array.isArray(items) ? items : []);
      if (canCreateOrders(current)) {
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

  function loadOrders(page = ordersPage.page, linkedOrderNumber = '', activeFilters = filtersRef.current, limit = ordersPage.limit) {
    return run(async () => {
      const result = await fetchOrders(page, limit, activeFilters);
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
      categoryID: '',
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
      // Тесто разных типов заявок не суммируется — выбор ограничен одним типом.
      const picked = orders.find((order) => order.number === number);
      const first = orders.find((order) => current.includes(order.number));
      if (picked && first && (picked.category?.id || null) !== (first.category?.id || null)) {
        setError('Выбирайте заявки одного типа. Сбросьте выбор, чтобы сменить тип.');
        return current;
      }
      setError('');
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

  // saveProduction создаёт документ отработки в журнале и перечитывает
  // выбранные заказы, чтобы лист сразу показал сохранённый факт.
  function saveProduction(ordersPayload) {
    return run(async () => {
      const sheet = await createProductionSheet(ordersPayload);
      logInfo('production.saved', { sheet: sheet.id, orders: ordersPayload.length });
      const loaded = await Promise.all(selectedOrderNumbers.map((number) => fetchOrder(number)));
      setSelectedOrders(loaded);
    });
  }

  function openSelectionReview() {
    return openOrdersSelection(selectedOrderNumbers);
  }

  function openOrdersSelection(numbers) {
    return run(async () => {
      const loaded = await Promise.all(numbers.map((number) => fetchOrder(number)));
      setSelectedOrderNumbers(numbers);
      setSelectedOrders(loaded);
      setSelectionMode(false);
      setMonitor(null);
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

  function changeCancellation(number, cancel) {
    return run(async () => {
      const saved = cancel ? await cancelOrder(number) : await restoreOrder(number);
      setSelectedOrder((current) => (current && current.number === number ? saved : current));
      setOrders((current) => current.map((o) => (o.number === number ? { ...o, cancelled: saved.cancelled } : o)));
    });
  }

  const canFavorite = viewer?.role === 'admin';
  const canManageOrders = canCreateOrders(viewer);
  const canUseMonitor = viewer?.role === 'baker' || viewer?.role === 'admin';
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

  if (canUseMonitor && route.name === 'production') {
    return (
      <OrdersLayout viewer={viewer} active={route.name} onNavigate={navigate} onLogout={handleLogout}>
        <ProductionJournal
          key={route.sheetId || 'list'}
          initialSheetId={route.sheetId}
          onOpenOrder={(number) => navigate({ name: 'orderView', number })}
        />
      </OrdersLayout>
    );
  }

  if (canUseMonitor && route.name === 'ordersTable') {
    return (
      <OrdersLayout viewer={viewer} active={route.name} onNavigate={navigate} onLogout={handleLogout}>
        <OrdersTableView
          catalog={catalog}
          onStartProduction={openOrdersSelection}
          onOpenProduction={(sheetId) => navigate({ name: 'production', sheetId })}
        />
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
          catalog={catalog}
          monitor={monitor}
          error={error}
          onBack={() => navigate({ name: 'orders' })}
          onRemove={removeSelectedOrder}
          onCalculate={calculateSelectedOrders}
          onSaveProduction={saveProduction}
          onOpenJournal={() => navigate({ name: 'production' })}
          onOpenProduction={(sheetId) => navigate({ name: 'production', sheetId })}
        />
      ) : canUseMonitor && (route.name === 'orderView' || route.name === 'orderMonitor') ? (
        <BakerOrderReview
          loading={loading}
          order={selectedOrder}
          catalog={catalog}
          monitor={monitor}
          error={error}
          canFavorite={canFavorite}
          onToggleFavorite={toggleFavorite}
          onBack={() => navigate({ name: 'orders' })}
          onCalculate={calculateCurrentOrder}
          onOpenProduction={(sheetId) => navigate({ name: 'production', sheetId })}
        />
      ) : canUseMonitor ? (
        <BakerOrdersView
          loading={loading}
          orders={orders}
          shops={shops}
          categories={categories}
          catalog={catalog}
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
          onShiftWindow={shiftMatrixWindow}
          onFiltersChange={updateFilters}
          onOpenProduction={(sheetId) => navigate({ name: 'production', sheetId })}
        />
      ) : (
        <ShopOrdersView
          loading={loading}
          orders={orders}
          page={ordersPage}
          shops={shops}
          categories={categories}
          viewer={viewer}
          filters={filters}
          selectedNumber={selectedNumber}
          selectedOrder={selectedOrder}
          editor={activeEditor}
          catalog={catalog}
          error={error}
          showCreateOrderPage={showCreateOrderPage}
          canFavorite={canFavorite}
          canManage={canManageOrders}
          onToggleFavorite={toggleFavorite}
          onCancelOrder={(number) => changeCancellation(number, true)}
          onRestoreOrder={(number) => changeCancellation(number, false)}
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
