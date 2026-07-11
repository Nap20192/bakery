import { useState } from 'react';
import { Button } from '../../ui/Button';
import { EmptyState } from '../../ui/EmptyState';
import { ErrorBanner } from '../../ui/ErrorBanner';
import { Icon } from '../../ui/Icon';
import { panelClass } from '../../ui/Panel';
import { categoryStyle } from '../../lib/categories';
import { formatQuantity } from '../../lib/format';
import { useOrdersTable } from './useOrdersTable';
import { ProductionColumnModal } from './ProductionColumnModal';

// Шапка липкая — фоны колонок сплошные, чтобы строки не просвечивали.
const toneClasses = {
  today: { header: 'bg-emerald-100 text-emerald-900', cell: 'bg-emerald-50/70', total: 'bg-emerald-100', badge: 'сегодня' },
  tomorrow: { header: 'bg-amber-100 text-amber-900', cell: 'bg-amber-50/70', total: 'bg-amber-100', badge: 'завтра' },
  '': { header: 'bg-white text-stone-600', cell: '', total: 'bg-stone-50', badge: '' },
};

// CategoryTable — таблица «блюда × даты» одного типа заявки. Раскладка под
// телефон: скролл-контейнер с «заморозкой» — колонка блюд липнет влево,
// строка дат — вверх, «Итого» — вниз; даты двигаются по горизонтали.
function CategoryTable({ group, columns, onOpenColumn }) {
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
                  <button
                    type="button"
                    onClick={() => onOpenColumn(column)}
                    className="min-h-11 w-full rounded-md px-1 hover:bg-black/5 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-stone-900/30"
                    aria-label={`Открыть заказы на ${column.label}`}
                  >
                    <span className="block text-[13px] font-semibold tabular-nums leading-5">{column.label}</span>
                    <span className="block text-[10px] font-medium uppercase leading-4 opacity-80">
                      {toneClasses[column.tone].badge || column.weekday}
                    </span>
                  </button>
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
export function OrdersTableView({ catalog = [], onOpenProduction, onStartProduction }) {
  const { activeGroup, columns, error, groups, loading, orders, setActiveCategoryID, setShift, shift } = useOrdersTable(catalog);
  const [openColumn, setOpenColumn] = useState(null);

  function ordersForColumn(column) {
    const categoryID = activeGroup?.category?.id || 0;
    return orders.filter((order) =>
      !order.cancelled
      && order.fulfillment_date === column.key
      && (order.category?.id || 0) === categoryID,
    );
  }

  function handleOpenColumn(column) {
    const columnOrders = ordersForColumn(column);
    if (!columnOrders.length) return;
    const sheetIDs = [...new Set(columnOrders.map((order) => order.production_sheet_id).filter(Boolean))];
    if (columnOrders.every((order) => order.production_sheet_id) && sheetIDs.length === 1) {
      onOpenProduction(sheetIDs[0]);
      return;
    }
    setOpenColumn({ column, orders: columnOrders });
  }

  return (
    <section className="px-3 py-3 pb-20 sm:px-5 sm:pb-5 lg:px-6">
      <div className="mx-auto max-w-[1440px] space-y-3">
        <div className="flex flex-wrap items-center justify-between gap-2">
          <div>
            <h1 className="m-0 text-lg font-semibold">Таблица</h1>
            <p className="m-0 text-[12px] leading-5 text-stone-500">Сколько печь по дням. Нажмите дату, чтобы отработать всю колонку.</p>
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
            {activeGroup && (
              <CategoryTable
                key={activeGroup.category?.id || 'other'}
                group={activeGroup}
                columns={columns}
                onOpenColumn={handleOpenColumn}
              />
            )}
          </>
        )}
      </div>
      {openColumn && (
        <ProductionColumnModal
          column={openColumn.column}
          category={activeGroup?.category}
          orders={openColumn.orders}
          onClose={() => setOpenColumn(null)}
          onOpenProduction={(sheetID) => {
            setOpenColumn(null);
            onOpenProduction(sheetID);
          }}
          onStartProduction={(numbers) => {
            setOpenColumn(null);
            onStartProduction(numbers);
          }}
        />
      )}
    </section>
  );
}
