import { Dialog, VisuallyHidden } from 'radix-ui';

// Modal — единый оверлей приложения поверх Radix Dialog: портал, focus trap,
// scroll lock, Escape и клик по фону закрывают, aria — из коробки.
// Содержимое и шапку отдаёт вызывающая сторона; API прежний (onClose).
export function Modal({ onClose, children, maxWidthClass = 'max-w-3xl' }) {
  return (
    <Dialog.Root open onOpenChange={(open) => { if (!open) onClose(); }}>
      <Dialog.Portal>
        {/* Content внутри Overlay — паттерн «скроллируемый оверлей»: длинное
            содержимое прокручивается вместе с подложкой. */}
        <Dialog.Overlay className="fade-in fixed inset-0 z-40 flex items-start justify-center overflow-y-auto bg-black/40 p-2 backdrop-blur-sm sm:p-4">
          <Dialog.Content
            className={`pop-in my-auto w-full ${maxWidthClass} rounded-xl border border-border bg-card p-4 text-card-foreground shadow-xl focus:outline-none`}
            aria-describedby={undefined}
            onOpenAutoFocus={(event) => event.preventDefault()}
          >
            <VisuallyHidden.Root asChild>
              <Dialog.Title>Диалоговое окно</Dialog.Title>
            </VisuallyHidden.Root>
            {children}
          </Dialog.Content>
        </Dialog.Overlay>
      </Dialog.Portal>
    </Dialog.Root>
  );
}
