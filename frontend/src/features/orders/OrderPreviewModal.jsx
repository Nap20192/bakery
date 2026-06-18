import { Button } from '../../components/Button';
import { Icon } from '../../components/Icon';
import { OrderDetails } from './OrderDetails';

export function OrderPreviewModal({ order, onClose }) {
  return (
    <div className="fixed inset-0 z-40 bg-black/35 p-2 sm:p-3">
      <div className="mx-auto max-h-[96vh] max-w-3xl overflow-y-auto rounded-lg border border-stone-300 bg-white p-3 shadow-xl">
        <div className="mb-3 flex justify-end">
          <Button onClick={onClose}>
            <Icon name="close" size={16} />
            Закрыть
          </Button>
        </div>
        <OrderDetails order={order} />
      </div>
    </div>
  );
}
