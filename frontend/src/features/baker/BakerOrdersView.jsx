import { useMemo, useState } from "react";
import { Button } from "../../ui/Button";
import { EmptyState } from "../../ui/EmptyState";
import { Icon } from "../../ui/Icon";
import { formatFulfillmentDate } from "../../lib/format";
import { BakerOrderCard } from "./BakerOrderCard";
import { BakerOrderFilters } from "./BakerOrderFilters";
import { OrderPreviewModal } from "../orders/OrderPreviewModal";
import { buildOrderMatrix } from "./orderMatrix";

export function BakerOrdersView({
  loading,
  orders,
  shops,
  categories = [],
  catalog = [],
  filters,
  selectedNumber,
  selectedOrderNumbers,
  error,
  selectionMode,
  canFavorite,
  onToggleFavorite,
  onLoadOrder,
  onSelect,
  onToggleSelection,
  onToggleSelectionMode,
  onOpenSelection,
  onShiftWindow,
  onFiltersChange,
  onOpenProduction,
}) {
  const [previewOpen, setPreviewOpen] = useState(false);
  const [previewOrder, setPreviewOrder] = useState(null);
  const selectedCount = selectedOrderNumbers.length;

  // Матрица «магазины × даты»: колонки — магазины, горизонтальные полосы —
  // даты выполнения (сначала самые дальние предстоящие). Внутри полосы каждая
  // колонка держит заявки своего магазина — высоты колонок неравномерны.
  const matrix = useMemo(() => buildOrderMatrix(orders, shops), [orders, shops]);

  function openPreview(order) {
    setPreviewOrder(order);
    setPreviewOpen(true);
    // List rows carry no history; fetch the full order so the preview shows
    // change statuses and the audit trail.
    if (onLoadOrder) {
      onLoadOrder(order.number)
        .then((full) => setPreviewOrder((cur) => (cur && cur.number === full.number ? full : cur)))
        .catch(() => {});
    }
  }

  return (
    <section className="px-3 py-3 pb-40 sm:px-5 sm:pb-24 lg:px-6" data-testid="bakerOrders-matrix">
      <div className="mx-auto max-w-[1440px] space-y-3">
        <BakerOrderFilters
          filters={filters}
          categories={categories}
          selectionMode={selectionMode}
          onToggleSelectionMode={onToggleSelectionMode}
          onFiltersChange={onFiltersChange}
        />

        {error && (
          <div className="rounded-md border border-red-200 bg-red-50 px-3 py-2 text-[13px] text-red-800">
            {error}
          </div>
        )}

        <div className="space-y-2">
          <div className="sticky top-[54px] z-10 bg-flour pb-1">
            <div
              className="grid gap-x-1.5 px-2 sm:gap-x-3"
              style={{ gridTemplateColumns: `repeat(${matrix.columns.length}, minmax(0, 1fr))` }}
            >
              {matrix.columns.map((column) => (
                <h3
                  key={column.key}
                  className="m-0 truncate rounded-md bg-stone-200/80 px-2 py-1.5 text-center text-[11px] font-semibold uppercase tracking-wide text-stone-700 sm:text-[12px]"
                  title={column.name}
                >
                  {column.name}
                </h3>
              ))}
            </div>
          </div>
          {matrix.bands.map((band) => (
            <DateBand key={band.key} band={band} columns={matrix.columns}>
              {matrix.columns.map((column) => (
                <div key={`${band.key}-${column.key}`} className="space-y-1.5">
                  {(band.cells.get(column.key) || []).map((order) => (
                    <BakerOrderCard
                      key={order.number}
                      order={order}
                      selected={selectedNumber === order.number}
                      checked={selectedOrderNumbers.includes(order.number)}
                      selectionMode={selectionMode}
                      onToggleSelection={() => onToggleSelection(order.number)}
                      onPreview={() => openPreview(order)}
                    />
                  ))}
                </div>
              ))}
            </DateBand>
          ))}
          {matrix.bands.length === 0 && <EmptyState compact>Заказов на эти дни нет.</EmptyState>}
        </div>

        <WindowNav loading={loading} filters={filters} onShiftWindow={onShiftWindow} />

        <BottomActionBar
          loading={loading}
          selectedCount={selectedCount}
          selectionMode={selectionMode}
          onOpenSelection={onOpenSelection}
        />

        {previewOpen && previewOrder && (
          <OrderPreviewModal
            order={previewOrder}
            catalog={catalog}
            canFavorite={canFavorite}
            onOpenProduction={onOpenProduction}
            onToggleFavorite={(number, fav) => {
              onToggleFavorite?.(number, fav);
              setPreviewOrder((cur) => (cur ? { ...cur, favorite: fav } : cur));
            }}
            onOpenOrder={(number) => {
              setPreviewOpen(false);
              onSelect(number);
            }}
            onClose={() => setPreviewOpen(false)}
          />
        )}
      </div>
    </section>
  );
}

// WindowNav walks the 5-day fulfillment window: later dates live «выше» in the
// matrix, so «Позже» loads the next window and «Раньше» the previous one.
function WindowNav({ loading, filters, onShiftWindow }) {
  if (!onShiftWindow || !filters.fulfillmentFrom) return null;
  return (
    <div className="grid grid-cols-[1fr_auto_1fr] items-center gap-2">
      <Button onClick={() => onShiftWindow(-5)} disabled={loading}>
        <Icon name="chevronLeft" size={15} />
        Раньше
      </Button>
      <span className="text-center text-[12px] leading-5 text-stone-500">
        {formatFulfillmentDate(filters.fulfillmentFrom)} — {formatFulfillmentDate(filters.fulfillmentTo)}
      </span>
      <Button onClick={() => onShiftWindow(5)} disabled={loading}>
        Позже
        <Icon name="chevronRight" size={15} />
      </Button>
    </div>
  );
}

// DateBand wraps a whole date section (header + cards). «Завтра» и «Сегодня» —
// главные рабочие полосы пекаря: вся секция подсвечивается мягкой цветной
// подложкой (янтарная — завтра, зелёная — сегодня), карточки на ней читаются
// контрастнее. Остальные даты — без подложки, с тонкой линией.
const bandStyles = {
  tomorrow: {
    panel: 'rounded-xl border border-amber-300/70 bg-amber-100/60 shadow-sm',
    label: 'text-amber-900',
    count: 'text-amber-700',
  },
  today: {
    panel: 'rounded-xl border border-emerald-300/70 bg-emerald-100/60 shadow-sm',
    label: 'text-emerald-900',
    count: 'text-emerald-700',
  },
};

function DateBand({ band, columns, children }) {
  const style = bandStyles[band.kind];
  const grid = (
    <div
      className="grid gap-x-1.5 gap-y-1.5 sm:gap-x-3"
      style={{ gridTemplateColumns: `repeat(${columns.length}, minmax(0, 1fr))` }}
    >
      {children}
    </div>
  );
  if (style) {
    return (
      <section className={`p-2 ${style.panel}`}>
        <div className="mb-1.5 flex items-baseline justify-between gap-2 px-0.5">
          <span className={`text-[13px] font-bold leading-5 ${style.label}`}>{band.label}</span>
          <span className={`text-[11px] font-semibold leading-5 tabular-nums ${style.count}`}>{band.total} заяв.</span>
        </div>
        {grid}
      </section>
    );
  }
  return (
    <section className="p-2">
      <div className="mb-1.5 flex items-center gap-2">
        <span className="shrink-0 text-[12px] font-semibold leading-5 text-stone-500">{band.label}</span>
        <span className="h-px flex-1 bg-stone-300" aria-hidden="true" />
        <span className="shrink-0 text-[11px] leading-5 text-stone-400 tabular-nums">{band.total}</span>
      </div>
      {grid}
    </section>
  );
}

function BottomActionBar({ loading, selectedCount, selectionMode, onOpenSelection }) {
  if (!selectionMode) {
    return null;
  }
  return (
    <div className="fade-in fixed inset-x-0 bottom-16 z-20 border-t border-stone-300 bg-white/95 px-3 py-2 backdrop-blur sm:bottom-0" data-testid="bakerOrders-selectionBar">
      <div className="mx-auto flex max-w-[1440px] items-center justify-between gap-2">
        <span className="min-w-0 truncate text-[13px] font-medium text-stone-700">Выбрано: {selectedCount}</span>
        <Button variant="primary" onClick={onOpenSelection} disabled={selectedCount === 0 || loading} className="shrink-0">
          Открыть выбранные
        </Button>
      </div>
    </div>
  );
}
