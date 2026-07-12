import { NavMenu } from '../../ui/NavMenu';

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

// OrdersLayout — единый навбар: шапка с логотипом и бургер-меню справа
// (drawer со списком разделов на всех вьюпортах). Нижней панели больше нет.
export function OrdersLayout({ viewer, active, onNavigate, onLogout, children }) {
  const role = navigationRole(viewer);
  const items = navItems.filter((item) => item.roles.includes(role));

  return (
    <div className="min-h-screen bg-flour text-stone-900">
      <a
        href="#main-content"
        className="fixed left-3 top-3 z-50 -translate-y-20 rounded-md bg-primary px-3 py-2 font-semibold text-primary-foreground shadow-lg transition-transform focus-visible:translate-y-0 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2"
      >
        К содержанию
      </a>
      <header className="sticky top-0 z-20 border-b border-border bg-card/95 px-3 py-2 backdrop-blur">
        <div className="mx-auto flex max-w-[1180px] items-center justify-between gap-2">
          <button
            type="button"
            onClick={() => onNavigate({ name: 'orders' })}
            className="inline-flex min-h-11 shrink-0 items-center rounded-md px-1 text-left text-title font-semibold tracking-tight text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring/50 sm:min-h-9"
            aria-label="Пекарня — к заказам"
          >
            Пекарня<span className="text-primary">.</span>
          </button>
          <NavMenu items={items} activeName={activeNavItem(active)} onNavigate={onNavigate} onLogout={onLogout} />
        </div>
      </header>
      <main id="main-content" tabIndex={-1} className="outline-none" data-testid="ordersLayout-mainContent">
        {children}
      </main>
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
