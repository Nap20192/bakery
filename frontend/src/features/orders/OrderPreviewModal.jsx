import { useEffect } from 'react';
import { Button } from '../../components/Button';
import { Icon } from '../../components/Icon';
import { OrderDetails } from './OrderDetails';

export function OrderPreviewModal({ order, onClose }) {
  useEffect(() => {
    const onKey = (event) => {
      if (event.key === 'Escape') onClose();
    };
    document.addEventListener('keydown', onKey);
    return () => document.removeEventListener('keydown', onKey);
  }, [onClose]);

  return (
    <div
      className="fade-in fixed inset-0 z-40 flex items-start justify-center overflow-y-auto bg-black/40 p-2 backdrop-blur-sm sm:p-4"
      onClick={onClose}
      role="presentation"
    >
      <div
        className="pop-in my-auto w-full max-w-3xl rounded-xl border border-stone-200 bg-white p-4 shadow-xl"
        onClick={(event) => event.stopPropagation()}
        role="dialog"
        aria-modal="true"
      >
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
