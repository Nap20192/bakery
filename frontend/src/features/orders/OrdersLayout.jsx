import { isWebMode } from '../../lib/auth';
import { Icon } from '../../components/Icon';
import { Button } from '../../components/Button';

const mobileColsClass = { 1: 'grid-cols-1', 2: 'grid-cols-2', 3: 'grid-cols-3', 4: 'grid-cols-4' };

const navItems = [
  { route: { name: 'orders' }, label: 'Заказы', icon: 'orders', roles: ['shop', 'baker', 'admin'] },
  { route: { name: 'orderNew' }, label: 'Новый', icon: 'plus', roles: ['shop'] },
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
    <main className={`min-h-screen bg-[#f7f3ea] text-stone-900 ${hideBottomNav ? '' : 'pb-16 sm:pb-0'}`}>
      <header className="sticky top-0 z-20 border-b border-stone-300 bg-white/95 px-3 py-2 shadow-sm backdrop-blur">
        <div className="mx-auto flex max-w-[1440px] items-center gap-2">
          <button
            type="button"
            onClick={() => onNavigate({ name: 'orders' })}
            className="shrink-0 rounded-md px-1 text-left text-[17px] font-semibold leading-6 text-stone-950 focus:outline-none focus:ring-2 focus:ring-stone-900/20"
          >
            Bakery
          </button>
          <nav className="hidden min-w-0 flex-1 gap-1 overflow-x-auto px-1 sm:flex">
            {items.map((item) => (
              <button
                key={item.label}
                type="button"
                onClick={() => onNavigate(item.route)}
                className={`shrink-0 rounded-md border px-3 py-1.5 text-[13px] font-semibold transition focus:outline-none focus:ring-2 focus:ring-stone-900/20 ${
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
      {children}
      {!hideBottomNav && (
      <nav className="fixed inset-x-0 bottom-0 z-30 border-t border-stone-300 bg-white/95 px-2 py-1.5 shadow-[0_-8px_20px_rgba(28,25,23,0.08)] backdrop-blur sm:hidden">
        <div className={`mx-auto grid max-w-md gap-1 ${mobileColsClass[items.length] ?? 'grid-cols-3'}`}>
          {items.map((item) => (
            <button
              key={item.label}
              type="button"
              onClick={() => onNavigate(item.route)}
              className={`flex min-h-12 flex-col items-center justify-center gap-0.5 rounded-md text-[11px] font-semibold transition focus:outline-none focus:ring-2 focus:ring-stone-900/20 ${
                activeNavItem(active) === item.route.name ? 'bg-stone-900 text-white' : 'text-stone-700 hover:bg-stone-50'
              }`}
            >
              <Icon name={item.icon} size={19} />
              <span className="truncate">{item.label}</span>
            </button>
          ))}
        </div>
      </nav>
      )}
    </main>
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
