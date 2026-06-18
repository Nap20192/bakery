import { Button } from '../../components/Button';
import { OrderDetails } from './OrderDetails';

export function OrderPreviewModal({ order, onClose }) {
  return (
    <div className="fixed inset-0 z-30 bg-black/35 p-3">
      <div className="mx-auto max-h-[92vh] max-w-3xl overflow-y-auto rounded-lg border border-stone-300 bg-white p-3 shadow-xl">
        <div className="mb-3 flex justify-end">
          <Button onClick={onClose}>Закрыть</Button>
        </div>
        <OrderDetails order={order} />
      </div>
    </div>
  );
}
