import { AlertDialog } from 'radix-ui';
import { Button } from './Button';

// ConfirmDialog — подтверждение деструктивного действия поверх Radix
// AlertDialog (замена window.confirm): focus trap, Escape = отмена, фокус
// приходит на «Отмена», клик мимо не закрывает (алерт требует выбора).
export function ConfirmDialog({ open, title, description = '', confirmLabel = 'Удалить', onConfirm, onCancel, busy = false }) {
  return (
    <AlertDialog.Root open={open} onOpenChange={(next) => { if (!next) onCancel(); }}>
      <AlertDialog.Portal>
        <AlertDialog.Overlay className="fade-in fixed inset-0 z-50 bg-slate-dark/60" />
        <AlertDialog.Content className="pop-in fixed left-1/2 top-1/2 z-50 w-[calc(100vw-2rem)] max-w-sm -translate-x-1/2 -translate-y-1/2 rounded-[1.5rem] border border-border bg-card p-6 text-card-foreground shadow-none focus:outline-none">
          <AlertDialog.Title className="m-0 text-title font-semibold leading-6 text-foreground">{title}</AlertDialog.Title>
          {description ? (
            <AlertDialog.Description className="m-0 mt-1 text-body leading-5 text-muted-foreground">{description}</AlertDialog.Description>
          ) : null}
          <div className="mt-4 flex justify-end gap-2">
            <AlertDialog.Cancel asChild>
              <Button disabled={busy}>Отмена</Button>
            </AlertDialog.Cancel>
            <AlertDialog.Action asChild>
              <Button variant="danger" disabled={busy} onClick={onConfirm}>
                {confirmLabel}
              </Button>
            </AlertDialog.Action>
          </div>
        </AlertDialog.Content>
      </AlertDialog.Portal>
    </AlertDialog.Root>
  );
}
