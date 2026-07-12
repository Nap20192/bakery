import { useState } from 'react';
import { Dialog } from 'radix-ui';
import { Icon } from './Icon';
import { isWebMode } from '../lib/auth';

// NavMenu — бургер-навигация: одна кнопка в шапке открывает правый drawer со
// списком разделов. Единый навбар для всех вьюпортов (заменяет прежние
// пилюли сверху и нижнюю панель). По DESIGN §4 — без пилюль: активный раздел
// отмечен терракотовой левой полосой и акцентным текстом, hover — вторичная
// поверхность.
export function NavMenu({ items, activeName, onNavigate, onLogout }) {
  const [open, setOpen] = useState(false);

  function go(route) {
    setOpen(false);
    onNavigate(route);
  }

  return (
    <Dialog.Root open={open} onOpenChange={setOpen}>
      <Dialog.Trigger asChild>
        <button
          type="button"
          data-testid="ordersLayout-menuButton"
          aria-label="Меню"
          className="inline-flex min-h-11 min-w-11 items-center justify-center rounded-md border border-border bg-card text-foreground transition hover:bg-secondary focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring/50 sm:min-h-9 sm:min-w-9"
        >
          <Icon name="menu" size={22} strokeWidth={1.75} />
        </button>
      </Dialog.Trigger>

      <Dialog.Portal>
        <Dialog.Overlay className="fade-in fixed inset-0 z-40 bg-inverse/50" />
        <Dialog.Content
          data-testid="ordersLayout-navDrawer"
          className="slide-in-right fixed inset-y-0 right-0 z-50 flex w-[min(20rem,86vw)] flex-col border-l border-border bg-card text-card-foreground shadow-[0_8px_24px_rgba(25,24,23,0.08)] focus:outline-none"
          aria-describedby={undefined}
        >
          <div className="flex items-center justify-between border-b border-border px-4 py-3">
            <Dialog.Title className="m-0 text-title font-semibold text-foreground">Меню</Dialog.Title>
            <Dialog.Close asChild>
              <button
                type="button"
                aria-label="Закрыть меню"
                className="inline-flex min-h-9 min-w-9 items-center justify-center rounded-md text-muted-foreground transition hover:bg-secondary hover:text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring/50"
              >
                <Icon name="close" size={20} />
              </button>
            </Dialog.Close>
          </div>

          <nav aria-label="Основная навигация" className="flex-1 overflow-y-auto p-2">
            {items.map((item) => {
              const active = activeName === item.route.name;
              return (
                <button
                  key={item.label}
                  type="button"
                  onClick={() => go(item.route)}
                  data-testid={`ordersLayout-navItem-${item.route.name}`}
                  aria-current={active ? 'page' : undefined}
                  className={`flex w-full items-center gap-3 rounded-md border-l-2 px-3 py-3 text-left text-body font-medium transition focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring/50 ${
                    active
                      ? 'border-l-primary bg-secondary text-primary'
                      : 'border-l-transparent text-foreground hover:bg-secondary'
                  }`}
                >
                  <Icon name={item.icon} size={19} className={active ? 'text-primary' : 'text-muted-foreground'} />
                  {item.label}
                </button>
              );
            })}
          </nav>

          {isWebMode() && (
            <div className="border-t border-border p-2">
              <button
                type="button"
                onClick={() => { setOpen(false); onLogout(); }}
                className="flex w-full items-center gap-3 rounded-md px-3 py-3 text-left text-body font-medium text-destructive transition hover:bg-destructive/10 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring/50"
              >
                <Icon name="logout" size={19} />
                Выйти
              </button>
            </div>
          )}
        </Dialog.Content>
      </Dialog.Portal>
    </Dialog.Root>
  );
}
