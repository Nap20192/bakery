import { useEffect } from 'react';

// Modal — единый оверлей приложения: затемнение, панель, Escape и клик по
// фону закрывают. Содержимое и шапку отдаёт вызывающая сторона.
export function Modal({ onClose, children, maxWidthClass = 'max-w-3xl' }) {
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
        className={`pop-in my-auto w-full ${maxWidthClass} rounded-xl border border-stone-200 bg-white p-4 shadow-xl`}
        onClick={(event) => event.stopPropagation()}
        role="dialog"
        aria-modal="true"
      >
        {children}
      </div>
    </div>
  );
}
