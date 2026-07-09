import { Button } from '../../ui/Button';
import { ErrorBanner } from '../../ui/ErrorBanner';
import { EmptyState } from '../../ui/EmptyState';
import { panelClass } from '../../ui/Panel';
import { OrderDetails } from '../orders/OrderDetails';
import { OrderEditor } from './OrderEditor';
import { OrderList } from './OrderList';

export function ShopOrdersView({
  loading,
  orders,
  page,
  shops,
  categories,
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
          <ErrorBanner error={error} className="mb-4" />
          {editor ? (
            <section className={panelClass}>
              <OrderEditor
                key={`${editor.mode}-${editor.order?.number || 'new'}`}
                catalog={catalog}
                categories={categories}
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
              <OrderDetails order={selectedOrder} catalog={catalog} canFavorite={canFavorite} onToggleFavorite={onToggleFavorite} />
            </section>
          ) : (
            <EmptyState>Заказы не загружены.</EmptyState>
          )}
        </div>
      </section>
    </div>
  );
}
