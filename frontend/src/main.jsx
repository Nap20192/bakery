import React, { useEffect, useMemo, useState } from 'react';
import { createRoot } from 'react-dom/client';
import './styles.css';

const apiBase = import.meta.env.VITE_API_BASE_URL || '/api';

const buttonClass =
  'inline-flex min-h-9 items-center justify-center rounded-lg border border-stone-300 bg-white px-3 py-2 text-sm font-medium text-slate-800 transition hover:border-stone-400 hover:bg-stone-50 disabled:cursor-not-allowed disabled:opacity-50';
const primaryButtonClass =
  'inline-flex min-h-9 items-center justify-center rounded-lg border border-slate-950 bg-slate-950 px-3 py-2 text-sm font-medium text-white transition hover:bg-slate-800 disabled:cursor-not-allowed disabled:opacity-50';
const panelClass = 'rounded-xl border border-stone-200 bg-white p-3 sm:p-4';

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
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');
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

  function loadMonitor() {
    if (!selectedOrder) return;
    return run(async () => {
      const result = await request(`/monitor/${encodeURIComponent(selectedOrder.number)}`);
      setMonitor(result);
    });
  }

  return (
    <main className="min-h-screen bg-[#f5f5f2] text-slate-900 lg:grid lg:grid-cols-[22rem_minmax(0,1fr)]">
      <aside className="flex max-h-[46vh] flex-col gap-3 border-b border-stone-200 bg-white p-3 lg:sticky lg:top-0 lg:h-screen lg:max-h-none lg:border-b-0 lg:border-r lg:p-4">
        <div className="flex items-center justify-between gap-3">
          <div className="flex min-w-0 items-center gap-3">
            <div className="min-w-0">
              <h1 className="m-0 text-lg font-semibold leading-tight">Заказы</h1>
              <span className="text-xs text-slate-500">Bakery</span>
            </div>
          </div>
        </div>

        <button className={primaryButtonClass} onClick={() => loadOrders(ordersPage.page)} disabled={loading}>
          Обновить
        </button>

        <div className="min-h-0 flex-1 space-y-2 overflow-y-auto pr-1">
          {orders.length ? (
            orders.map((order) => (
              <button
                key={order.number}
                className={`w-full rounded-lg border px-3 py-2 text-left transition ${
                  selectedOrder?.number === order.number
                    ? 'border-slate-950 bg-stone-50'
                    : 'border-transparent bg-white hover:border-stone-300 hover:bg-stone-50'
                }`}
                onClick={() => {
                  loadOrder(order.number);
                }}
              >
                <div className="grid grid-cols-[minmax(0,1fr)_auto] gap-2">
                  <strong className="truncate text-sm font-semibold text-slate-900">{order.number}</strong>
                  <span className="text-xs text-slate-500">{order.items?.length || 0} поз.</span>
                  <span className="truncate text-xs text-slate-600">Откуда: {orderSource(order)}</span>
                  <span className="text-xs text-slate-500">{formatFulfillmentDate(order.fulfillment_date) || '-'}</span>
                </div>
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

      <section className="min-w-0 p-3 sm:p-5 lg:p-6">
        <header className="mb-3 sm:mb-4">
          <div className="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
            <div className="min-w-0">
              <span className="text-[11px] font-medium uppercase text-slate-500">Просмотр заказа</span>
              <h2 className="m-0 truncate text-lg font-semibold leading-tight text-slate-950 sm:text-2xl">{pageTitle}</h2>
            </div>
            <div className="grid grid-cols-2 gap-2 sm:flex">
              <button className={buttonClass} onClick={() => selectedOrder && loadOrder(selectedOrder.number)} disabled={!selectedOrder || loading}>
                Обновить заказ
              </button>
              <button className={buttonClass} onClick={() => loadMonitor()} disabled={!selectedOrder || loading}>
                Расчёт теста
              </button>
            </div>
          </div>
        </header>

        {error && <div className="mb-4 rounded-md border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-800">{error}</div>}

        {selectedOrder ? (
          <div className="grid gap-4 xl:grid-cols-[minmax(0,1.15fr)_minmax(24rem,0.85fr)]">
            <section className={panelClass}>
              <OrderDetails order={selectedOrder} />
            </section>

            <section className={panelClass}>
              <PanelHeader eyebrow="Расход полуфабрикатов" title="Расчёт теста" />
              <button className={buttonClass} onClick={() => loadMonitor()} disabled={!selectedOrder || loading}>
                Рассчитать
              </button>
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
    <div className="overflow-hidden rounded-lg border border-stone-200 bg-white">
      <div className="hidden grid-cols-[5rem_minmax(0,1fr)_5rem_5rem] gap-2 bg-stone-50 px-3 py-2 text-xs font-medium uppercase text-slate-500 sm:grid">
        <span>Код</span>
        <span>Позиция</span>
        <span className="text-right">Кол-во</span>
        <span className="text-right">Всего</span>
      </div>
      <div className="divide-y divide-stone-200">
        {items.map((item) => (
          <div
            className="grid grid-cols-[4.2rem_minmax(0,1fr)_3.5rem] items-center gap-2 px-3 py-2 text-sm sm:grid-cols-[5rem_minmax(0,1fr)_5rem_5rem]"
            key={item.code}
          >
            <code className="truncate text-xs text-slate-500">{item.code}</code>
            <span className="min-w-0 truncate leading-5 text-slate-800" title={item.product_name}>
              {item.product_name}
            </span>
            <span className="hidden text-right text-sm text-slate-600 sm:block">{orderQuantity(item)}</span>
            <strong className="truncate text-right text-xs font-semibold text-slate-950 sm:text-sm">
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
    return <EmptyState compact>Нажмите "Рассчитать", чтобы увидеть расход теста.</EmptyState>;
  }
  if (!monitor.reports?.length) {
    return <EmptyState compact>Нет данных для расчёта теста.</EmptyState>;
  }
  return (
    <div className="mt-3 overflow-hidden rounded-lg border border-stone-200 bg-white">
      {monitor.reports.map(({ code, report }) => (
        <article className="border-b border-stone-200 last:border-b-0" key={code}>
          <header className="grid grid-cols-[minmax(0,1fr)_auto] gap-2 bg-stone-50 px-3 py-2">
            <strong className="min-w-0 truncate text-sm text-slate-950" title={report.ingredient.product_name}>
              {report.ingredient.product_name}
            </strong>
            <span className="text-sm font-semibold text-slate-950">
              {formatQuantity(report.ingredient.quantity)} {report.ingredient.unit}
            </span>
          </header>
          <div className="divide-y divide-stone-200">
            {report.breakdown?.map((item) => (
              <div
                className="grid grid-cols-[4.2rem_minmax(0,1fr)_5.5rem] items-center gap-2 px-3 py-2 text-sm sm:grid-cols-[5rem_minmax(0,1fr)_8rem]"
                key={`${code}-${item.order_item_code}`}
              >
                <code className="truncate text-xs text-slate-500">{item.order_item_code}</code>
                <span className="min-w-0 truncate text-slate-700" title={item.order_item_name}>
                  {item.order_item_name}
                </span>
                <strong className="truncate text-right text-xs font-semibold text-slate-950 sm:text-sm">
                  {formatQuantity(item.ingredient_quantity)} {report.ingredient.unit}
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
