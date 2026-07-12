import { Button } from '../../ui/Button';
import { Icon } from '../../ui/Icon';
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
  onDuplicate,
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
                mode={editor.mode}
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
              {/* Единая панель действий заказа: все кнопки в один ряд, справа,
                  одной высоты. «Дублировать» создаёт новый заказ, копируя
                  позиции источника. */}
              <div className="mb-3 flex flex-wrap items-center justify-end gap-2">
                {canManage && (
                  <>
                    <Button
                      onClick={() => onDuplicate?.(selectedOrder.number)}
                      disabled={loading}
                      title="Создать новый заказ, скопировав позиции этого"
                    >
                      <Icon name="copy" size={15} />
                      Дублировать
                    </Button>
                    {selectedOrder.cancelled ? (
                      <Button onClick={() => onRestoreOrder?.(selectedOrder.number)} disabled={loading}>
                        Восстановить
                      </Button>
                    ) : (
                      <Button variant="danger" onClick={() => onCancelOrder?.(selectedOrder.number)} disabled={loading}>
                        <Icon name="close" size={15} />
                        Отменить
                      </Button>
                    )}
                    <Button variant="primary" onClick={onEdit} disabled={loading || selectedOrder.cancelled}>
                      Изменить
                    </Button>
                  </>
                )}
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
