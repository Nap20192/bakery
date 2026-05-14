import React, { useEffect, useMemo, useState } from 'react';
import { createRoot } from 'react-dom/client';
import { Activity, ClipboardList, Menu, RefreshCcw, Search, X } from 'lucide-react';
import './styles.css';

const apiBase = import.meta.env.VITE_API_BASE_URL || '/api';

const buttonClass =
  'inline-flex min-h-9 items-center justify-center gap-2 rounded-xl border border-stone-300 bg-white px-3 py-2 text-sm font-medium text-slate-800 transition hover:border-stone-400 hover:bg-stone-50 disabled:cursor-not-allowed disabled:opacity-50';
const primaryButtonClass =
  'inline-flex min-h-9 items-center justify-center gap-2 rounded-xl border border-slate-950 bg-slate-950 px-3 py-2 text-sm font-medium text-white transition hover:bg-slate-800 disabled:cursor-not-allowed disabled:opacity-50';
const iconButtonClass =
  'inline-flex h-10 w-10 shrink-0 items-center justify-center rounded-xl border border-stone-300 bg-white text-slate-800 transition hover:border-stone-400 hover:bg-stone-50';
const panelClass = 'rounded-2xl border border-stone-200 bg-white p-3 shadow-sm sm:p-4';

function formatQuantity(value) {
  if (!Number.isFinite(value)) return '0';
  return new Intl.NumberFormat('ru-RU', { maximumFractionDigits: 3 }).format(value);
}

function formatDate(value) {
  if (!value) return '';
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return value;
  return new Intl.DateTimeFormat('ru-RU', { dateStyle: 'medium', timeStyle: 'short' }).format(date);
}

function formatFulfillmentDate(value) {
  if (!value) return '';
  const date = new Date(`${value}T00:00:00`);
  if (Number.isNaN(date.getTime())) return value;
  return new Intl.DateTimeFormat('ru-RU', { dateStyle: 'medium' }).format(date);
}

function orderQuantity(item) {
  if (!item.reserved_quantity) return formatQuantity(item.quantity);
  return `${formatQuantity(item.quantity)}+${formatQuantity(item.reserved_quantity)}`;
}

function orderDepartmentName(order, key) {
  return order?.[key]?.name || '-';
}

function orderSource(order) {
  return order?.from_department?.name || order?.location || '-';
}

function orderCreator(order) {
  if (order?.created_by_username) return `@${order.created_by_username}`;
  return orderSource(order);
}

function apiURL(base, path) {
  return `${base.replace(/\/$/, '')}${path}`;
}

function App() {
  const [orders, setOrders] = useState([]);
  const [selectedOrder, setSelectedOrder] = useState(null);
  const [monitor, setMonitor] = useState(null);
  const [monitorCode, setMonitorCode] = useState('');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');
  const [menuOpen, setMenuOpen] = useState(false);
  const [ordersPage, setOrdersPage] = useState({
    page: 1,
    limit: 10,
    total: 0,
    total_pages: 0,
  });

  const selectedNumber = selectedOrder?.number || '';
  const pageTitle = useMemo(() => selectedNumber || 'Последние заказы', [selectedNumber]);

  useEffect(() => {
    loadOrders();
  }, []);

  async function request(path) {
    const response = await fetch(apiURL(apiBase, path));
    const payload = await response.json().catch(() => ({}));
    if (!response.ok) {
      throw new Error(payload.error || `HTTP ${response.status}`);
    }
    return payload;
  }

  async function run(task) {
    setLoading(true);
    setError('');
    try {
      await task();
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setLoading(false);
    }
  }

  function loadOrders(page = ordersPage.page) {
    return run(async () => {
      const result = await request(`/orders?page=${page}&limit=${ordersPage.limit}`);
      const items = result.items || [];
      setOrders(items);
      setOrdersPage({
        page: result.page || page,
        limit: result.limit || ordersPage.limit,
        total: result.total || 0,
        total_pages: result.total_pages || 0,
      });
      if (!selectedOrder && items.length > 0) {
        setSelectedOrder(items[0]);
      }
    });
  }

  function loadOrder(number) {
    return run(async () => {
      const order = await request(`/orders/${encodeURIComponent(number)}`);
      setSelectedOrder(order);
      setMonitor(null);
    });
  }

  function loadMonitor(code = '') {
    if (!selectedOrder) return;
    return run(async () => {
      const suffix = code.trim() ? `/${encodeURIComponent(code.trim())}` : '';
      const result = await request(`/monitor/${encodeURIComponent(selectedOrder.number)}${suffix}`);
      setMonitor(result);
    });
  }

  return (
    <main className="min-h-screen bg-[#f3f4f0] text-slate-900 lg:grid lg:grid-cols-[20rem_minmax(0,1fr)]">
      {menuOpen && (
        <button
          className="fixed inset-0 z-30 bg-slate-950/25 backdrop-blur-[1px] lg:hidden"
          type="button"
          aria-label="Закрыть меню"
          onClick={() => setMenuOpen(false)}
        />
      )}

      <aside
        className={`fixed bottom-3 left-3 top-3 z-40 flex w-[min(22rem,calc(100vw-1.5rem))] flex-col gap-3 overflow-hidden rounded-3xl border border-white/80 bg-[#f7f8f5] p-3 shadow-2xl transition-transform duration-200 lg:sticky lg:top-0 lg:bottom-auto lg:left-auto lg:z-auto lg:h-screen lg:w-auto lg:rounded-none lg:border-r lg:border-stone-200 lg:bg-white lg:p-4 lg:shadow-none ${
          menuOpen ? 'translate-x-0' : '-translate-x-full'
        }`}
      >
        <div className="flex items-center justify-between gap-3">
          <div className="flex min-w-0 items-center gap-3">
            <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-2xl bg-slate-950 text-white">
              <ClipboardList size={21} />
            </div>
            <div className="min-w-0">
              <h1 className="m-0 text-lg font-semibold leading-tight">Bakery</h1>
              <span className="text-xs text-slate-500">Orders</span>
            </div>
          </div>
          <button className={`${iconButtonClass} lg:hidden`} type="button" aria-label="Закрыть меню" onClick={() => setMenuOpen(false)}>
            <X size={19} />
          </button>
        </div>

        <button className={primaryButtonClass} onClick={() => loadOrders(ordersPage.page)} disabled={loading}>
          <RefreshCcw size={16} />
          Обновить
        </button>

        <div className="min-h-0 flex-1 space-y-2 overflow-y-auto pr-1">
          {orders.length ? (
            orders.map((order) => (
              <button
                key={order.number}
                className={`w-full rounded-2xl border p-3 text-left transition ${
                  selectedOrder?.number === order.number
                    ? 'border-slate-950 bg-slate-950 text-white'
                    : 'border-transparent bg-white hover:border-stone-300 hover:bg-stone-50'
                }`}
                onClick={() => {
                  setMenuOpen(false);
                  loadOrder(order.number);
                }}
              >
                <strong className={`block break-words text-sm font-semibold ${selectedOrder?.number === order.number ? 'text-white' : 'text-slate-900'}`}>
                  {order.number}
                </strong>
                <span className={`mt-1 block break-words text-xs leading-5 ${selectedOrder?.number === order.number ? 'text-stone-200' : 'text-slate-600'}`}>
                  От кого: {orderCreator(order)}
                </span>
                <span className={`block break-words text-xs leading-5 ${selectedOrder?.number === order.number ? 'text-stone-200' : 'text-slate-600'}`}>
                  Откуда: {orderSource(order)}
                </span>
                <span className={`block break-words text-xs leading-5 ${selectedOrder?.number === order.number ? 'text-stone-200' : 'text-slate-600'}`}>
                  Куда: {orderDepartmentName(order, 'to_department')}
                </span>
                <span className={`block break-words text-xs leading-5 ${selectedOrder?.number === order.number ? 'text-stone-200' : 'text-slate-600'}`}>
                  Выполнить: {formatFulfillmentDate(order.fulfillment_date) || '-'}
                </span>
                <span className={`mt-1 block break-words text-xs ${selectedOrder?.number === order.number ? 'text-stone-300' : 'text-slate-500'}`}>
                  Создан: {formatDate(order.created_at) || '-'} · {order.items?.length || 0} поз.
                </span>
              </button>
            ))
          ) : (
            <EmptyState compact>Заказов нет.</EmptyState>
          )}
        </div>

        <div className="grid grid-cols-[1fr_auto_1fr] items-center gap-2">
          <button className={buttonClass} onClick={() => loadOrders(ordersPage.page - 1)} disabled={loading || ordersPage.page <= 1}>
            Назад
          </button>
          <span className="whitespace-nowrap text-xs text-slate-500">
            {ordersPage.page} / {ordersPage.total_pages || 1}
          </span>
          <button
            className={buttonClass}
            onClick={() => loadOrders(ordersPage.page + 1)}
            disabled={loading || ordersPage.page >= (ordersPage.total_pages || 1)}
          >
            Далее
          </button>
        </div>
      </aside>

      <section className="min-w-0 p-2.5 sm:p-5 lg:p-6">
        <header className="sticky top-0 z-20 -mx-2.5 mb-3 border-b border-stone-200/70 bg-[#f3f4f0]/95 px-2.5 py-2 backdrop-blur sm:static sm:mx-0 sm:mb-4 sm:border-0 sm:bg-transparent sm:p-0 sm:backdrop-blur-none">
          <div className="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
          <div className="flex min-w-0 items-center gap-3">
            <button className={`${iconButtonClass} lg:hidden`} type="button" aria-label="Открыть заказы" onClick={() => setMenuOpen(true)}>
              <Menu size={19} />
            </button>
            <div className="min-w-0">
              <span className="text-[11px] font-medium uppercase text-slate-500">Просмотр заказа</span>
              <h2 className="m-0 break-words text-lg font-semibold leading-tight text-slate-950 sm:text-2xl">{pageTitle}</h2>
            </div>
          </div>
          <div className="grid grid-cols-2 gap-2 sm:flex">
            <button className={buttonClass} onClick={() => selectedOrder && loadOrder(selectedOrder.number)} disabled={!selectedOrder || loading}>
              <RefreshCcw size={16} />
              Заказ
            </button>
            <button className={buttonClass} onClick={() => loadMonitor()} disabled={!selectedOrder || loading}>
              <Activity size={16} />
              Мониторинг
            </button>
          </div>
          </div>
        </header>

        {error && <div className="mb-4 rounded-md border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-800">{error}</div>}

        {selectedOrder ? (
          <div className="grid gap-4 xl:grid-cols-[minmax(0,1.15fr)_minmax(22rem,0.85fr)]">
            <section className={panelClass}>
              <OrderDetails order={selectedOrder} />
            </section>

            <section className={panelClass}>
              <PanelHeader eyebrow="Расход ингредиентов" title="Мониторинг" />
              <div className="grid gap-2 sm:grid-cols-[minmax(0,1fr)_auto]">
                <input
                  className="min-h-10 min-w-0 rounded-xl border border-stone-300 bg-white px-3 py-2 text-sm text-slate-900 outline-none transition focus:border-slate-950 focus:ring-2 focus:ring-slate-950/10"
                  placeholder="Код ингредиента"
                  value={monitorCode}
                  onChange={(event) => setMonitorCode(event.target.value)}
                />
                <button className={buttonClass} onClick={() => loadMonitor(monitorCode)} disabled={loading}>
                  <Search size={16} />
                  Найти
                </button>
              </div>
              <MonitorReports monitor={monitor} />
            </section>
          </div>
        ) : (
          <EmptyState>Заказы не загружены.</EmptyState>
        )}
      </section>
    </main>
  );
}

function PanelHeader({ eyebrow, title, count }) {
  return (
    <div className="mb-3 flex items-start justify-between gap-3">
      <div className="min-w-0">
        <span className="text-[11px] font-medium uppercase text-slate-500">{eyebrow}</span>
        <h3 className="m-0 break-words text-base font-semibold leading-tight text-slate-950 sm:text-lg">{title}</h3>
      </div>
      {count !== undefined && (
        <span className="shrink-0 rounded-full border border-stone-200 bg-stone-50 px-2 py-1 text-[11px] font-medium text-slate-600 sm:text-xs">
          {count} поз.
        </span>
      )}
    </div>
  );
}

function OrderDetails({ order }) {
  return (
    <>
      <PanelHeader eyebrow="Заказ" title={order.number} count={order.items?.length || 0} />

      <div className="mb-3 grid grid-cols-2 gap-1.5 sm:gap-2 md:grid-cols-5">
        <MetaCell label="Создан" value={formatDate(order.created_at) || '-'} />
        <MetaCell label="Выполнить" value={formatFulfillmentDate(order.fulfillment_date) || '-'} />
        <MetaCell label="От кого" value={orderCreator(order)} />
        <MetaCell label="Откуда" value={order.from_department?.name || order.location || '-'} />
        <MetaCell label="Куда" value={order.to_department?.name || '-'} />
      </div>

      <OrderItems items={order.items || []} />
    </>
  );
}

function MetaCell({ label, value }) {
  return (
    <div className="min-w-0 rounded-xl border border-stone-200 bg-stone-50 px-2.5 py-2 sm:px-3">
      <span className="block text-[11px] text-slate-500">{label}</span>
      <strong className="block break-words text-xs font-semibold leading-5 text-slate-900 sm:text-sm">{value}</strong>
    </div>
  );
}

function OrderItems({ items }) {
  if (!items.length) {
    return <EmptyState compact>В заказе нет позиций.</EmptyState>;
  }
  return (
    <div className="overflow-hidden rounded-2xl border border-stone-200 bg-white">
      <div className="hidden grid-cols-[5rem_minmax(0,1fr)_5rem_5rem] gap-2 bg-stone-50 px-3 py-2 text-xs font-semibold uppercase text-slate-500 sm:grid">
        <span>Код</span>
        <span>Позиция</span>
        <span className="text-right">Кол-во</span>
        <span className="text-right">Всего</span>
      </div>
      <div className="divide-y divide-stone-200">
        {items.map((item) => (
          <div
            className="grid grid-cols-[4.4rem_minmax(0,1fr)] gap-x-2 gap-y-1 px-3 py-2.5 text-sm sm:grid-cols-[5rem_minmax(0,1fr)_5rem_5rem] sm:items-start sm:py-2"
            key={item.code}
          >
            <code className="self-start break-words rounded-lg bg-stone-100 px-1.5 py-0.5 text-xs text-slate-700">{item.code}</code>
            <span className="min-w-0 break-words leading-5 text-slate-800">{item.product_name}</span>
            <span className="col-start-2 text-xs text-slate-500 sm:hidden">
              Кол-во: {orderQuantity(item)}
            </span>
            <span className="hidden text-right text-sm text-slate-600 sm:block">{orderQuantity(item)}</span>
            <strong className="col-start-2 text-xs font-semibold text-slate-950 sm:col-start-auto sm:text-right sm:text-sm">
              <span className="sm:hidden">Всего: </span>
              {formatQuantity(item.production_quantity)}
            </strong>
          </div>
        ))}
      </div>
    </div>
  );
}

function MonitorReports({ monitor }) {
  if (!monitor) {
    return <EmptyState compact>Нажмите "Мониторинг" для расчета по дефолтным кодам.</EmptyState>;
  }
  if (!monitor.reports?.length) {
    return <EmptyState compact>Нет данных мониторинга.</EmptyState>;
  }
  return (
    <div className="mt-3 space-y-3">
      {monitor.reports.map(({ code, report }) => (
        <article className="rounded-2xl border border-stone-200 bg-white" key={code}>
          <header className="flex flex-col gap-1 border-b border-stone-200 bg-stone-50 px-3 py-2 sm:flex-row sm:items-center sm:justify-between">
            <strong className="min-w-0 break-words text-sm text-slate-950">
              <code className="rounded-lg bg-white px-1.5 py-0.5 text-xs text-slate-700">{report.ingredient.product_code}</code>{' '}
              {report.ingredient.product_name}
            </strong>
            <span className="text-sm font-semibold text-slate-950">
              {formatQuantity(report.ingredient.quantity)} {report.ingredient.unit}
            </span>
          </header>
          <div className="divide-y divide-stone-200">
            {report.breakdown?.map((item) => (
              <div
                className="grid grid-cols-1 gap-1 px-3 py-2 text-sm sm:grid-cols-[minmax(0,1fr)_auto] sm:gap-2"
                key={`${code}-${item.order_item_code}`}
              >
                <span className="min-w-0 break-words text-slate-700">
                  <code className="rounded-lg bg-stone-100 px-1.5 py-0.5 text-xs text-slate-700">{item.order_item_code}</code>{' '}
                  {item.order_item_name}
                </span>
                <strong className="text-left font-semibold text-slate-950 sm:text-right">
                  {formatQuantity(item.order_item_quantity)} / {formatQuantity(item.ingredient_quantity)} {report.ingredient.unit}
                </strong>
              </div>
            ))}
          </div>
        </article>
      ))}
    </div>
  );
}

function EmptyState({ children, compact = false }) {
  return (
    <div className={`rounded-md border border-dashed border-stone-300 bg-stone-50 text-center text-sm text-slate-500 ${compact ? 'p-3' : 'p-6'}`}>
      {children}
    </div>
  );
}

createRoot(document.getElementById('root')).render(<App />);
