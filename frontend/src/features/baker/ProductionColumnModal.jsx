import { useMemo, useState } from 'react';
import { Button } from '../../ui/Button';
import { CategoryBadge } from '../../ui/CategoryBadge';
import { Modal } from '../../ui/Modal';
import { SheetBadge } from '../../ui/SheetBadge';
import { formatFulfillmentDate } from '../../lib/format';
import { orderSource } from '../../lib/orders';
import { productionDeviations } from '../../lib/production';

// ProductionColumnModal turns one date column into a production batch.
// Unworked orders are selected by default; worked orders link to their sheet.
export function ProductionColumnModal({ column, category, orders, onClose, onOpenProduction, onStartProduction }) {
  const available = useMemo(() => orders.filter((order) => !order.production_sheet_id), [orders]);
  const [selected, setSelected] = useState(() => available.map((order) => order.number));
  const selectedSet = useMemo(() => new Set(selected), [selected]);

  function toggle(number) {
    setSelected((current) => current.includes(number)
      ? current.filter((item) => item !== number)
      : [...current, number]);
  }

  return (
    <Modal onClose={onClose} maxWidthClass="max-w-xl">
      <div className="mb-4 flex items-start justify-between gap-3">
        <div>
          <span className="text-[11px] font-medium uppercase tracking-wide text-stone-500">Партия на дату</span>
          <h2 className="m-0 text-[20px] font-semibold leading-7 text-stone-950">
            {formatFulfillmentDate(column.key)}
          </h2>
          <div className="mt-1"><CategoryBadge category={category} /></div>
        </div>
        <Button variant="ghost" onClick={onClose}>Закрыть</Button>
      </div>

      <div className="mb-3 flex items-center justify-between gap-3 rounded-lg border border-stone-200 bg-stone-50 px-3 py-2">
        <div>
          <strong className="block text-[13px] text-stone-900">Заказов: {orders.length}</strong>
          <span className="text-[12px] text-stone-600">
            К отработке: {available.length} · уже отработано: {orders.length - available.length}
          </span>
        </div>
        {available.length > 0 && (
          <button
            type="button"
            className="shrink-0 text-[12px] font-semibold text-stone-700 underline decoration-stone-300 underline-offset-2"
            onClick={() => setSelected(selected.length === available.length ? [] : available.map((order) => order.number))}
          >
            {selected.length === available.length ? 'Снять все' : 'Выбрать все'}
          </button>
        )}
      </div>

      <div className="max-h-[55vh] space-y-2 overflow-y-auto pr-1" data-testid="productionColumn-orderList">
        {orders.map((order) => {
          const worked = Boolean(order.production_sheet_id);
          if (worked) {
            return (
              <button
                type="button"
                key={order.number}
                onClick={() => onOpenProduction(order.production_sheet_id)}
                className="flex w-full items-center gap-3 rounded-lg border border-stone-300 bg-stone-100 px-3 py-2 text-left transition hover:border-stone-500 hover:bg-stone-200 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-stone-900/30"
              >
                <span className="inline-flex h-5 w-5 shrink-0 items-center justify-center rounded-full bg-stone-700 text-[11px] font-bold text-white" aria-hidden="true">✓</span>
                <span className="min-w-0 flex-1">
                  <strong className="block truncate text-[13px] text-stone-950">{order.number}</strong>
                  <span className="block truncate text-[12px] text-stone-600">{orderSource(order)} · {order.items?.length || 0} поз.</span>
                </span>
                <SheetBadge sheetId={order.production_sheet_id} deviations={productionDeviations(order)} showStatus />
              </button>
            );
          }
          return (
            <label
              key={order.number}
              className="flex cursor-pointer items-center gap-3 rounded-lg border border-stone-200 bg-white px-3 py-2 transition hover:border-stone-400"
            >
              <input
                type="checkbox"
                checked={selectedSet.has(order.number)}
                onChange={() => toggle(order.number)}
                aria-label={`Выбрать заказ ${order.number}`}
                className="h-5 w-5 shrink-0 accent-stone-900"
              />
              <div className="min-w-0 flex-1">
                <strong className="block truncate text-[13px] text-stone-950">{order.number}</strong>
                <span className="block truncate text-[12px] text-stone-600">{orderSource(order)} · {order.items?.length || 0} поз.</span>
              </div>
            </label>
          );
        })}
      </div>

      <div className="mt-4 flex flex-wrap justify-end gap-2 border-t border-stone-200 pt-3">
        <Button onClick={onClose}>Отмена</Button>
        <Button
          variant="primary"
          disabled={selected.length === 0}
          onClick={() => onStartProduction(selected)}
          data-testid="productionColumn-startButton"
        >
          Отработать {selected.length > 0 ? `${selected.length} заказ.` : ''}
        </Button>
      </div>
    </Modal>
  );
}
