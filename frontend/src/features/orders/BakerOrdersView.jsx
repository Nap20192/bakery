import { useState } from 'react';
import { Button } from '../../components/Button';
import { EmptyState } from '../../components/EmptyState';
import { BakerOrderCard } from './BakerOrderCard';
import { BakerOrderFilters } from './BakerOrderFilters';
import { OrderPagination } from './OrderPagination';
import { OrderPreviewModal } from './OrderPreviewModal';

export function BakerOrdersView({
  loading,
  orders,
  page,
  shops,
  filters,
  selectedNumber,
  selectedOrder,
  selectedOrderNumbers,
  error,
  selectionMode,
  onSelect,
  onToggleSelection,
  onToggleSelectionMode,
  onOpenSelection,
  onPageChange,
  onFiltersChange,
  onResetFilters,
}) {
  const [previewOpen, setPreviewOpen] = useState(false);
  const [previewOrder, setPreviewOrder] = useState(null);
  const selectedCount = selectedOrderNumbers.length;

  return (
    <section className="px-3 py-3 pb-40 sm:px-5 sm:pb-24 lg:px-6">
      <div className="mx-auto max-w-[1440px] space-y-4">
        <BakerOrderFilters
          filters={filters}
          shops={shops}
          selectionMode={selectionMode}
          onToggleSelectionMode={onToggleSelectionMode}
          onFiltersChange={onFiltersChange}
          onResetFilters={onResetFilters}
        />

        {error && <div className="rounded-md border border-red-200 bg-red-50 px-3 py-2 text-[13px] text-red-800">{error}</div>}

        <div className="grid gap-3 [grid-template-columns:repeat(auto-fill,minmax(13rem,1fr))]">
          {orders.length ? (
            orders.map((order) => (
              <BakerOrderCard
                key={order.number}
                order={order}
                selected={selectedNumber === order.number}
                checked={selectedOrderNumbers.includes(order.number)}
                selectionMode={selectionMode}
                onSelect={() => onSelect(order.number)}
                onToggleSelection={() => onToggleSelection(order.number)}
                onPreview={() => {
                  setPreviewOrder(order);
                  setPreviewOpen(true);
                }}
              />
            ))
          ) : (
            <div className="col-span-full">
              <EmptyState compact>Заказов нет.</EmptyState>
            </div>
          )}
        </div>

        <OrderPagination loading={loading} page={page} onPageChange={onPageChange} />

        <BottomActionBar
          loading={loading}
          selectedOrder={selectedOrder}
          selectedCount={selectedCount}
          selectionMode={selectionMode}
          onPreview={() => {
            setPreviewOrder(selectedOrder);
            setPreviewOpen(true);
          }}
          onOpenSelection={onOpenSelection}
        />

        {previewOpen && previewOrder && (
          <OrderPreviewModal order={previewOrder} onClose={() => setPreviewOpen(false)} />
        )}
      </div>
    </section>
  );
}

function BottomActionBar({ loading, selectedOrder, selectedCount, selectionMode, onPreview, onOpenSelection }) {
  if (!selectedOrder && selectedCount === 0) {
    return null;
  }
  return (
    <div className="fixed inset-x-0 bottom-16 z-20 border-t border-stone-300 bg-white/95 px-3 py-2 backdrop-blur sm:bottom-0">
      <div className="mx-auto flex max-w-[1440px] items-center justify-between gap-2">
        <span className="min-w-0 truncate text-[13px] font-medium text-stone-700">
          {selectionMode ? `Выбрано: ${selectedCount}` : selectedOrder?.number}
        </span>
        <div className="flex shrink-0 gap-2">
          <Button onClick={onPreview} disabled={!selectedOrder || loading}>
            Обзор
          </Button>
          {selectionMode && (
            <Button variant="primary" onClick={onOpenSelection} disabled={selectedCount === 0 || loading}>
              Открыть выбранные
            </Button>
          )}
        </div>
      </div>
    </div>
  );
}
