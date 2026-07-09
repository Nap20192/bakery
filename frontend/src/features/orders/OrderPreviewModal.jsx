import { Button } from '../../ui/Button';
import { Icon } from '../../ui/Icon';
import { Modal } from '../../ui/Modal';
import { OrderDetails } from './OrderDetails';

export function OrderPreviewModal({ order, catalog = [], onClose, onOpenOrder, onOpenProduction, canFavorite, onToggleFavorite }) {
  return (
    <Modal onClose={onClose}>
      <div className="mb-3 flex justify-end gap-2">
        {onOpenOrder && (
          <Button variant="primary" onClick={() => onOpenOrder(order.number)}>
            <Icon name="calculator" size={16} />
            Расчёт теста
          </Button>
        )}
        <Button onClick={onClose}>
          <Icon name="close" size={16} />
          Закрыть
        </Button>
      </div>
      <OrderDetails order={order} catalog={catalog} canFavorite={canFavorite} onToggleFavorite={onToggleFavorite} onOpenProduction={onOpenProduction} />
    </Modal>
  );
}
