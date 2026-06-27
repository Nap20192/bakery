import { Button } from '../../components/Button';
import { EmptyState } from '../../components/EmptyState';
import { panelClass } from '../../components/Panel';
import { OrderDetails } from './OrderDetails';
import { OrderEditor } from './OrderEditor';
import { OrderList } from './OrderList';

export function ShopOrdersView({
  loading,
  orders,
  page,
  shops,
  viewer,
  filters,
  selectedNumber,
  selectedOrder,
  editor,
  catalog,
  error,
  showCreateOrderPage,
  canFavorite,
  canManage,
  onToggleFavorite,
  onCancelOrder,
  onRestoreOrder,
  onSelect,
  onPageChange,
  onFiltersChange,
  onResetFilters,
  onEdit,
  onCancelEdit,
  onSave,
}) {
  return (
    <div className={showCreateOrderPage ? '' : 'lg:grid lg:grid-cols-[24rem_minmax(0,1fr)]'}>
      {!showCreateOrderPage && (
        <OrderList
          loading={loading}
          orders={orders}
          page={page}
          shops={shops}
          viewer={viewer}
          canFilterShops
          filters={filters}
          selectedNumber={selectedNumber}
          onSelect={onSelect}
          onPageChange={onPageChange}
          onFiltersChange={onFiltersChange}
          onResetFilters={onResetFilters}
        />
      )}
      <section className="min-w-0 p-3 pt-0 sm:p-5 lg:p-6">
        <div className={`mx-auto ${showCreateOrderPage ? 'max-w-5xl' : 'max-w-[1180px]'}`}>
          {error && <div className="mb-4 rounded-md border border-red-200 bg-red-50 px-3 py-2 text-[13px] text-red-800">{error}</div>}
          {editor ? (
            <section className={panelClass}>
              <OrderEditor
                key={`${editor.mode}-${editor.order?.number || 'new'}`}
                catalog={catalog}
                order={editor.order}
                shops={shops}
                loading={loading}
                onCancel={onCancelEdit}
                onSave={onSave}
              />
            </section>
          ) : selectedOrder ? (
            <section className={panelClass}>
              <div className="mb-3 flex flex-wrap justify-end gap-2">
                {canManage && (
                  selectedOrder.cancelled ? (
                    <Button onClick={() => onRestoreOrder?.(selectedOrder.number)} disabled={loading}>
                      Восстановить
                    </Button>
                  ) : (
                    <Button variant="danger" onClick={() => onCancelOrder?.(selectedOrder.number)} disabled={loading}>
                      Отменить
                    </Button>
                  )
                )}
                <Button onClick={onEdit} disabled={loading || selectedOrder.cancelled}>
                  Изменить
                </Button>
              </div>
              <OrderDetails order={selectedOrder} canFavorite={canFavorite} onToggleFavorite={onToggleFavorite} />
            </section>
          ) : (
            <EmptyState>Заказы не загружены.</EmptyState>
          )}
        </div>
      </section>
    </div>
  );
}
