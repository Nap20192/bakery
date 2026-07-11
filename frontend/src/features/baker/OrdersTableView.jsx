import { useEffect, useMemo, useState } from 'react';
import { Button } from '../../ui/Button';
import { EmptyState } from '../../ui/EmptyState';
import { ErrorBanner } from '../../ui/ErrorBanner';
import { Icon } from '../../ui/Icon';
import { panelClass } from '../../ui/Panel';
import { fetchCategories, fetchOrders } from '../../api/orders';
import { categoryStyle } from '../../lib/categories';
import { formatQuantity } from '../../lib/format';

// Окно таблицы: вчера + неделя вперёд, листается кнопками на неделю.
const WINDOW_DAYS = 8;
const WINDOW_START_OFFSET = -1;

const WEEKDAYS = ['вс', 'пн', 'вт', 'ср', 'чт', 'пт', 'сб'];

function dateKey(date) {
  const y = date.getFullYear();
  const m = String(date.getMonth() + 1).padStart(2, '0');
  const d = String(date.getDate()).padStart(2, '0');
  return `${y}-${m}-${d}`;
}

function addDays(date, days) {
  const next = new Date(date);
  next.setDate(next.getDate() + days);
  return next;
}

// Колонки окна: сегодня и завтра несут семантическую подсветку — те же цвета,
// что в матрице (зелёная — сегодня, янтарная — завтра).
function buildColumns(shift) {
  const today = new Date();
  today.setHours(0, 0, 0, 0);
  const start = addDays(today, WINDOW_START_OFFSET + shift * (WINDOW_DAYS - 1));
  const todayKey = dateKey(today);
  const tomorrowKey = dateKey(addDays(today, 1));
  return Array.from({ length: WINDOW_DAYS }, (_, index) => {
    const date = addDays(start, index);
    const key = dateKey(date);
    return {
      key,
      label: `${String(date.getDate()).padStart(2, '0')}.${String(date.getMonth() + 1).padStart(2, '0')}`,
      weekday: WEEKDAYS[date.getDay()],
      tone: key === todayKey ? 'today' : key === tomorrowKey ? 'tomorrow' : '',
    };
  });
}

// Шапка липкая — фоны колонок сплошные, чтобы строки не просвечивали.
const toneClasses = {
  today: { header: 'bg-emerald-100 text-emerald-900', cell: 'bg-emerald-50/70', total: 'bg-emerald-100', badge: 'сегодня' },
  tomorrow: { header: 'bg-amber-100 text-amber-900', cell: 'bg-amber-50/70', total: 'bg-amber-100', badge: 'завтра' },
  '': { header: 'bg-white text-stone-600', cell: '', total: 'bg-stone-50', badge: '' },
};

// buildGroups делит заказы окна по типу заявки (хлеб/булочки/…) и внутри
// каждого типа группирует по блюдам: строка — блюдо, ячейка — суммарное
// количество (заявка + резерв) на дату по всем магазинам.
function buildGroups(orders, columns, catalog, categories) {
  const columnKeys = new Set(columns.map((column) => column.key));
  const catalogIndex = new Map(catalog.map((dish, index) => [String(dish.name || '').toLowerCase().trim(), index]));
  const groups = new Map();
  for (const order of orders) {
    if (order.cancelled) continue;
    const date = String(order.fulfillment_date || '');
    if (!columnKeys.has(date)) continue;
    const categoryID = order.category?.id || 0;
    if (!groups.has(categoryID)) {
      groups.set(categoryID, { category: order.category || null, byDish: new Map() });
    }
    const { byDish } = groups.get(categoryID);
    for (const item of order.items || []) {
      const key = String(item.product_name || '').toLowerCase().trim();
      if (!byDish.has(key)) {
        byDish.set(key, { key, name: item.product_name, cells: {}, total: 0 });
      }
      const row = byDish.get(key);
      const quantity = Number(item.production_quantity || 0);
      row.cells[date] = (row.cells[date] || 0) + quantity;
      row.total += quantity;
    }
  }

  const byDishSort = (a, b) => {
    const ai = catalogIndex.has(a.key) ? catalogIndex.get(a.key) : Number.MAX_SAFE_INTEGER;
    const bi = catalogIndex.has(b.key) ? catalogIndex.get(b.key) : Number.MAX_SAFE_INTEGER;
    if (ai !== bi) return ai - bi;
    return a.name.localeCompare(b.name, 'ru');
  };
  // Секции — в порядке справочника типов; легаси-заказы без типа — в конец.
  const categoryOrder = new Map(categories.map((category, index) => [category.id, index]));
  return [...groups.entries()]
    .sort(([aID], [bID]) => (categoryOrder.get(aID) ?? Number.MAX_SAFE_INTEGER) - (categoryOrder.get(bID) ?? Number.MAX_SAFE_INTEGER))
    .map(([, group]) => ({
      category: group.category,
      rows: [...group.byDish.values()].sort(byDishSort),
    }));
}

// CategoryTable — таблица «блюда × даты» одного типа заявки. Раскладка под
// телефон: скролл-контейнер с «заморозкой» — колонка блюд липнет влево,
// строка дат — вверх, «Итого» — вниз; даты двигаются по горизонтали.
function CategoryTable({ group, columns }) {
  const totals = {};
  for (const column of columns) {
    totals[column.key] = group.rows.reduce((sum, row) => sum + (row.cells[column.key] || 0), 0);
  }
  return (
    <section className={`${panelClass} p-0 sm:p-0`}>
      <div className="max-h-[72vh] overflow-auto overscroll-contain">
        <table className="w-full min-w-[640px] border-collapse text-[13px]">
          <thead>
            <tr>
              <th className="sticky left-0 top-0 z-30 border-b border-stone-200 bg-white px-3 py-2 text-left text-[11px] font-medium uppercase text-stone-500">
                Блюдо
              </th>
              {columns.map((column) => (
                <th
                  key={column.key}
                  className={`sticky top-0 z-20 min-w-[3.5rem] border-b border-stone-200 px-2 py-2 text-center align-top ${toneClasses[column.tone].header}`}
                >
                  <span className="block text-[13px] font-semibold tabular-nums leading-5">{column.label}</span>
                  <span className="block text-[10px] font-medium uppercase leading-4 opacity-80">
                    {toneClasses[column.tone].badge || column.weekday}
                  </span>
                </th>
              ))}
            </tr>
          </thead>
          <tbody>
            {group.rows.map((row) => (
              <tr key={row.key} className="border-b border-stone-100 last:border-0">
                <td className="sticky left-0 z-10 max-w-[9.5rem] break-words bg-white px-3 py-1.5 text-[12px] leading-4 text-stone-800 sm:max-w-[16rem] sm:text-[13px] sm:leading-5">
                  {row.name}
                </td>
                {columns.map((column) => (
                  <td key={column.key} className={`px-2 py-1.5 text-center tabular-nums ${toneClasses[column.tone].cell} ${row.cells[column.key] ? 'font-medium text-stone-900' : 'text-stone-300'}`}>
                    {row.cells[column.key] ? formatQuantity(row.cells[column.key]) : '·'}
                  </td>
                ))}
              </tr>
            ))}
          </tbody>
          <tfoot>
            <tr>
              <td className="sticky bottom-0 left-0 z-30 border-t border-stone-300 bg-stone-50 px-3 py-2 text-[12px] font-semibold uppercase text-stone-600">
                Итого
              </td>
              {columns.map((column) => (
                <td
                  key={column.key}
                  className={`sticky bottom-0 z-20 border-t border-stone-300 px-2 py-2 text-center text-[13px] font-semibold tabular-nums text-stone-900 ${toneClasses[column.tone].total}`}
                >
                  {totals[column.key] ? formatQuantity(totals[column.key]) : '·'}
                </td>
              ))}
            </tr>
          </tfoot>
        </table>
      </div>
    </section>
  );
}

// OrdersTableView — режим «Таблица»: сводка «блюда × даты» по всем заказам,
// разбитая на секции по типу заявки (хлеб/булочки/…). Внутри секции строки —
// блюда в порядке каталога, колонки — даты окна, ячейка — сколько печь на
// день суммарно по всем магазинам. Сегодня/завтра выделены цветом.
export function OrdersTableView({ catalog = [] }) {
  const [shift, setShift] = useState(0);
  const [orders, setOrders] = useState([]);
  const [categories, setCategories] = useState([]);
  const [activeCategoryID, setActiveCategoryID] = useState(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');

  const columns = useMemo(() => buildColumns(shift), [shift]);

  useEffect(() => {
    fetchCategories()
      .then((rows) => setCategories(Array.isArray(rows) ? rows : []))
      .catch(() => setCategories([]));
  }, []);

  useEffect(() => {
    let cancelled = false;
    async function load() {
      setLoading(true);
      setError('');
      try {
        const filters = { fulfillmentFrom: columns[0].key, fulfillmentTo: columns[columns.length - 1].key };
        // Окно может содержать больше 100 заказов — выгребаем постранично.
        const all = [];
        for (let page = 1; page <= 10; page += 1) {
          const result = await fetchOrders(page, 100, filters);
          all.push(...(result.items || []));
          if (all.length >= (result.total || 0) || !(result.items || []).length) break;
        }
        if (!cancelled) setOrders(all);
      } catch (err) {
        if (!cancelled) setError(err instanceof Error ? err.message : String(err));
      } finally {
        if (!cancelled) setLoading(false);
      }
    }
    load();
    return () => { cancelled = true; };
  }, [columns]);

  const groups = useMemo(() => buildGroups(orders, columns, catalog, categories), [orders, columns, catalog, categories]);

  // Хлеб и булочки — на «одной странице», но по очереди: активен один тип,
  // остальные переключаются чипами. Выбор переживает смену окна дат, пока
  // тип присутствует в данных.
  const activeGroup = useMemo(() => {
    if (!groups.length) return null;
    return groups.find((group) => (group.category?.id || 0) === activeCategoryID) || groups[0];
  }, [groups, activeCategoryID]);

  return (
    <section className="px-3 py-3 pb-20 sm:px-5 sm:pb-5 lg:px-6">
      <div className="mx-auto max-w-[1440px] space-y-3">
        <div className="flex flex-wrap items-center justify-between gap-2">
          <div>
            <h1 className="m-0 text-lg font-semibold">Таблица</h1>
            <p className="m-0 text-[12px] leading-5 text-stone-500">Сколько печь по дням: все заказы, сгруппированные по блюдам.</p>
          </div>
          <div className="flex shrink-0 gap-1.5">
            <Button onClick={() => setShift((value) => value - 1)} disabled={loading} aria-label="Раньше">
              <Icon name="chevronLeft" size={16} />
              Раньше
            </Button>
            <Button onClick={() => setShift(0)} disabled={loading || shift === 0}>Сегодня</Button>
            <Button onClick={() => setShift((value) => value + 1)} disabled={loading} aria-label="Позже">
              Позже
              <Icon name="chevronRight" size={16} />
            </Button>
          </div>
        </div>

        <ErrorBanner error={error} />

        {groups.length === 0 && !loading ? (
          <EmptyState>Нет заказов на выбранные даты.</EmptyState>
        ) : (
          <>
            <div className="flex flex-wrap items-center gap-1.5">
              {groups.map((group) => {
                const id = group.category?.id || 0;
                const active = (activeGroup?.category?.id || 0) === id;
                const style = categoryStyle(group.category);
                return (
                  <button
                    key={id}
                    type="button"
                    onClick={() => setActiveCategoryID(id)}
                    className={`inline-flex min-h-9 items-center gap-1.5 rounded-full border px-3 py-1 text-[13px] font-semibold transition focus:outline-none focus:ring-2 focus:ring-stone-900/20 ${
                      active ? style.chipActive : 'border-stone-300 bg-white text-stone-600 hover:border-stone-400'
                    }`}
                  >
                    <span className={`h-2 w-2 rounded-full ${style.dot}`} aria-hidden="true" />
                    {group.category?.name || 'Без типа'}
                    <span className="text-[11px] font-medium opacity-70">{group.rows.length}</span>
                  </button>
                );
              })}
            </div>
            {activeGroup && <CategoryTable key={activeGroup.category?.id || 'other'} group={activeGroup} columns={columns} />}
          </>
        )}
      </div>
    </section>
  );
}
