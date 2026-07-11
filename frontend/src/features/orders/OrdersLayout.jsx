import { isWebMode } from '../../lib/auth';
import { Icon } from '../../ui/Icon';
import { Button } from '../../ui/Button';

const mobileColsClass = { 1: 'grid-cols-1', 2: 'grid-cols-2', 3: 'grid-cols-3', 4: 'grid-cols-4', 5: 'grid-cols-5', 6: 'grid-cols-6', 7: 'grid-cols-7' };

const navItems = [
  { route: { name: 'orders' }, label: 'Заказы', icon: 'orders', roles: ['shop', 'baker', 'admin'] },
  { route: { name: 'orderNew' }, label: 'Новый', icon: 'plus', roles: ['shop'] },
  { route: { name: 'ordersTable' }, label: 'Таблица', icon: 'table', roles: ['baker', 'admin'] },
  { route: { name: 'doughCalc' }, label: 'Тесто', icon: 'calculator', roles: ['baker', 'admin'] },
  { route: { name: 'production' }, label: 'Отработки', icon: 'journal', roles: ['baker', 'admin'] },
  { route: { name: 'adminUsers' }, label: 'Люди', icon: 'users', roles: ['admin'] },
  { route: { name: 'adminDishes' }, label: 'Блюда', icon: 'orders', roles: ['admin'] },
  { route: { name: 'me' }, label: 'Профиль', icon: 'user', roles: ['shop', 'baker', 'admin'] },
];

export function OrdersLayout({ viewer, active, onNavigate, onLogout, children }) {
  const role = navigationRole(viewer);
  const items = navItems.filter((item) => item.roles.includes(role));
  // While creating/editing an order the form owns the bottom bar — hide the
  // mobile bottom nav so the two don't stack.
  const hideBottomNav = active === 'orderNew' || active === 'orderEdit';

  return (
    <div className={`min-h-screen bg-flour text-stone-900 ${hideBottomNav ? '' : 'pb-[calc(4rem+env(safe-area-inset-bottom))] sm:pb-0'}`}>
      <a
        href="#main-content"
        className="fixed left-3 top-3 z-50 -translate-y-20 rounded-md bg-primary px-3 py-2 font-semibold text-primary-foreground shadow-lg transition-transform focus-visible:translate-y-0 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2"
      >
        К содержанию
      </a>
      <header className="sticky top-0 z-20 border-b border-stone-300 bg-white/95 px-3 py-2 shadow-sm backdrop-blur">
        <div className="mx-auto flex max-w-[1440px] items-center gap-2">
          <button
            type="button"
            onClick={() => onNavigate({ name: 'orders' })}
            className="inline-flex min-h-11 shrink-0 items-center rounded-md px-1 text-left text-page font-bold leading-6 tracking-tight text-stone-950 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring/50 sm:min-h-9"
            aria-label="Пекарня — к заказам"
          >
            Пекарня<span className="text-amber-500">.</span>
          </button>
          <nav aria-label="Основная навигация" data-testid="ordersLayout-desktopNav" className="hidden min-w-0 flex-1 gap-1 overflow-x-auto px-1 sm:flex">
            {items.map((item) => (
              <button
                key={item.label}
                type="button"
                onClick={() => onNavigate(item.route)}
                data-testid={`ordersLayout-navItem-${item.route.name}`}
                aria-current={activeNavItem(active) === item.route.name ? 'page' : undefined}
                className={`min-h-9 shrink-0 rounded-md border px-3 py-1.5 text-body font-semibold transition focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring/50 ${
                  activeNavItem(active) === item.route.name
                    ? 'border-stone-900 bg-stone-900 text-white'
                    : 'border-stone-300 bg-white text-stone-800'
                }`}
              >
                <Icon name={item.icon} size={16} className="mr-1 inline-block align-[-3px]" />
                {item.label}
              </button>
            ))}
          </nav>
          {isWebMode() && (
            <Button type="button" onClick={onLogout} variant="danger" className="shrink-0">
              <Icon name="logout" size={16} />
              Выйти
            </Button>
          )}
        </div>
      </header>
      <main id="main-content" tabIndex={-1} className="outline-none" data-testid="ordersLayout-mainContent">
        {children}
      </main>
      {!hideBottomNav && (
      <nav aria-label="Основная навигация" data-testid="ordersLayout-mobileNav" className="fixed inset-x-0 bottom-0 z-30 border-t border-stone-300 bg-white/95 px-2 pb-[calc(0.375rem+env(safe-area-inset-bottom))] pt-1.5 shadow-[0_-8px_20px_rgba(28,25,23,0.08)] backdrop-blur sm:hidden">
        <div className={`mx-auto grid max-w-md gap-1 ${mobileColsClass[items.length] ?? 'grid-cols-3'}`}>
          {items.map((item) => (
            <button
              key={item.label}
              type="button"
              onClick={() => onNavigate(item.route)}
              data-testid={`ordersLayout-mobileNavItem-${item.route.name}`}
              aria-current={activeNavItem(active) === item.route.name ? 'page' : undefined}
              className={`flex min-h-12 min-w-0 flex-col items-center justify-center gap-0.5 rounded-md px-1 text-caption font-semibold transition focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring/50 ${
                activeNavItem(active) === item.route.name ? 'bg-stone-900 text-white' : 'text-stone-700 hover:bg-stone-50'
              }`}
            >
              <Icon name={item.icon} size={19} />
              <span className="w-full truncate text-center max-[359px]:sr-only">{item.label}</span>
            </button>
          ))}
        </div>
      </nav>
      )}
    </div>
  );
}

function activeNavItem(active) {
  if (active === 'orderView' || active === 'orderEdit' || active === 'orderMonitor' || active === 'orderSelection') {
    return 'orders';
  }
  return active;
}

function navigationRole(viewer) {
  if (viewer?.role === 'admin') return 'admin';
  if (viewer?.department_type === 'shop' || viewer?.role === 'shop') return 'shop';
  if (viewer?.department_type === 'workshop' || viewer?.role === 'baker') return 'baker';
  return viewer?.role || '';
}
