import { isWebMode } from '../../lib/auth';
import { Icon } from '../../components/Icon';

const navItems = [
  { route: { name: 'orders' }, label: 'Заказы', icon: 'orders', roles: ['shop', 'baker', 'admin'] },
  { route: { name: 'orderNew' }, label: 'Новый', icon: 'plus', roles: ['shop'] },
  { route: { name: 'adminUsers' }, label: 'Люди', icon: 'users', roles: ['admin'] },
  { route: { name: 'me' }, label: 'Профиль', icon: 'user', roles: ['shop', 'baker', 'admin'] },
];

export function OrdersLayout({ viewer, active, onNavigate, onLogout, children }) {
  const role = navigationRole(viewer);
  const items = navItems.filter((item) => item.roles.includes(role));

  return (
    <main className="min-h-screen bg-[#f7f3ea] pb-16 text-stone-900 sm:pb-0">
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
            <button
              type="button"
              onClick={onLogout}
              className="inline-flex min-h-9 shrink-0 items-center gap-1.5 rounded-md border border-red-300 px-3 py-1.5 text-[13px] font-semibold text-red-700 transition hover:bg-red-50 focus:outline-none focus:ring-2 focus:ring-red-900/20"
            >
              <Icon name="logout" size={16} />
              Выйти
            </button>
          )}
        </div>
      </header>
      {children}
      <nav className="fixed inset-x-0 bottom-0 z-30 border-t border-stone-300 bg-white/95 px-2 py-1.5 shadow-[0_-8px_20px_rgba(28,25,23,0.08)] backdrop-blur sm:hidden">
        <div className="mx-auto grid max-w-md grid-cols-3 gap-1">
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
