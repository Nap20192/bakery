import React, { useEffect, useMemo, useState } from 'react';
import { createRoot } from 'react-dom/client';
import { Activity, ClipboardList, RefreshCcw, Search } from 'lucide-react';
import './styles.css';

const apiBase = import.meta.env.VITE_API_BASE_URL || '/api';

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

  function loadOrders() {
    return run(async () => {
      const result = await request('/orders?limit=50');
      setOrders(result);
      if (!selectedOrder && result.length > 0) {
        setSelectedOrder(result[0]);
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
    <main className="shell">
      <aside className="sidebar">
        <div className="brand">
          <ClipboardList size={22} />
          <div>
            <h1>Bakery</h1>
            <span>Orders</span>
          </div>
        </div>

        <button className="button primary" onClick={loadOrders} disabled={loading}>
          <RefreshCcw size={16} />
          Обновить
        </button>

        <div className="orders">
          {orders.map((order) => (
            <button
              key={order.number}
              className={`order-row ${selectedOrder?.number === order.number ? 'active' : ''}`}
              onClick={() => loadOrder(order.number)}
            >
              <strong>{order.number}</strong>
              <span>{order.items?.length || 0} поз.</span>
            </button>
          ))}
        </div>
      </aside>

      <section className="content">
        <header className="topbar">
          <div>
            <span className="eyebrow">Просмотр заказа</span>
            <h2>{pageTitle}</h2>
          </div>
          <div className="actions">
            <button className="button" onClick={() => selectedOrder && loadOrder(selectedOrder.number)} disabled={!selectedOrder || loading}>
              <RefreshCcw size={16} />
              Заказ
            </button>
            <button className="button" onClick={() => loadMonitor()} disabled={!selectedOrder || loading}>
              <Activity size={16} />
              Мониторинг
            </button>
          </div>
        </header>

        {error && <div className="alert">{error}</div>}

        {selectedOrder ? (
          <div className="grid">
            <section className="panel">
              <OrderDetails order={selectedOrder} />
            </section>

            <section className="panel">
              <div className="panel-header">
                <div>
                  <span className="eyebrow">Расход ингредиентов</span>
                  <h3>Мониторинг</h3>
                </div>
              </div>
              <div className="monitor-form">
                <input
                  placeholder="Код ингредиента"
                  value={monitorCode}
                  onChange={(event) => setMonitorCode(event.target.value)}
                />
                <button className="button" onClick={() => loadMonitor(monitorCode)} disabled={loading}>
                  <Search size={16} />
                  Найти
                </button>
              </div>
              <MonitorReports monitor={monitor} />
            </section>
          </div>
        ) : (
          <div className="empty">Заказы не загружены.</div>
        )}
      </section>
    </main>
  );
}

function OrderDetails({ order }) {
  return (
    <>
      <div className="panel-header">
        <div>
          <span className="eyebrow">Заказ</span>
          <h3>{order.number}</h3>
        </div>
        <code>{order.monitor_command}</code>
      </div>

      <div className="meta">
        <div>
          <span>Создан</span>
          <strong>{formatDate(order.created_at) || '-'}</strong>
        </div>
        <div>
          <span>Выполнить</span>
          <strong>{formatFulfillmentDate(order.fulfillment_date) || '-'}</strong>
        </div>
        <div>
          <span>Откуда</span>
          <strong>{order.from_department?.name || order.location || '-'}</strong>
        </div>
        <div>
          <span>Куда</span>
          <strong>{order.to_department?.name || '-'}</strong>
        </div>
      </div>

      <table>
        <thead>
          <tr>
            <th>Код</th>
            <th>Позиция</th>
            <th>Кол-во</th>
            <th>Всего</th>
          </tr>
        </thead>
        <tbody>
          {order.items?.map((item) => (
            <tr key={item.code}>
              <td data-label="Код"><code>{item.code}</code></td>
              <td data-label="Позиция">{item.product_name}</td>
              <td data-label="Кол-во">{orderQuantity(item)}</td>
              <td data-label="Всего">{formatQuantity(item.production_quantity)}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </>
  );
}

function MonitorReports({ monitor }) {
  if (!monitor) {
    return <div className="empty compact">Нажмите "Мониторинг" для расчета по дефолтным кодам.</div>;
  }
  if (!monitor.reports?.length) {
    return <div className="empty compact">Нет данных мониторинга.</div>;
  }
  return (
    <div className="reports">
      {monitor.reports.map(({ code, report }) => (
        <article className="report" key={code}>
          <header>
            <strong><code>{report.ingredient.product_code}</code> {report.ingredient.product_name}</strong>
            <span>{formatQuantity(report.ingredient.quantity)} {report.ingredient.unit}</span>
          </header>
          <div className="breakdown">
            {report.breakdown?.map((item) => (
              <div className="breakdown-row" key={`${code}-${item.order_item_code}`}>
                <span><code>{item.order_item_code}</code> {item.order_item_name}</span>
                <strong>{formatQuantity(item.order_item_quantity)} / {formatQuantity(item.ingredient_quantity)} {report.ingredient.unit}</strong>
              </div>
            ))}
          </div>
        </article>
      ))}
    </div>
  );
}

createRoot(document.getElementById('root')).render(<App />);
